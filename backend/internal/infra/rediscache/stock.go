package rediscache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	reserveOK       int64 = 1
	reserveOutStock int64 = -1
	reserveMiss     int64 = -2

	reservationExpirationsKey = "reservation:order:expirations"
)

var (
	ErrStockCacheMiss = errors.New("stock cache miss")
	ErrInsufficient   = errors.New("insufficient stock")
)

type StockItem struct {
	ProductID string `json:"product_id"`
	Qty       int    `json:"qty"`
}

type Reservation struct {
	OrderID   string      `json:"order_id"`
	ExpiresAt int64       `json:"expires_at"`
	Items     []StockItem `json:"items"`
}

// Rebuild replaces the projection, including reservation details. Callers must
// hold the shared database inventory lock so an in-flight Reserve is not lost.
func (s *StockStore) Rebuild(ctx context.Context, stocks map[string]int, reservations []Reservation) error {
	if !s.Enabled() {
		return nil
	}
	if reservations == nil {
		reservations = []Reservation{}
	}
	payload, err := json.Marshal(struct {
		Stocks       map[string]int `json:"stocks"`
		Reservations []Reservation  `json:"reservations"`
	}{stocks, reservations})
	if err != nil {
		return err
	}
	return rebuildScript.Run(ctx, s.client, []string{"stock:products", reservationExpirationsKey}, string(payload)).Err()
}

type StockStore struct {
	client *redis.Client
}

func NewStockStore(client *redis.Client) *StockStore {
	if client == nil {
		return nil
	}
	return &StockStore{client: client}
}

func (s *StockStore) Enabled() bool {
	return s != nil && s.client != nil
}

func (s *StockStore) PrimeStocks(ctx context.Context, stocks map[string]int) error {
	if !s.Enabled() || len(stocks) == 0 {
		return nil
	}

	pipe := s.client.Pipeline()
	for productID, stock := range stocks {
		pipe.SetNX(ctx, stockKey(productID), stock, 0)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (s *StockStore) Available(ctx context.Context, productID string) (int, bool, error) {
	if !s.Enabled() {
		return 0, false, nil
	}

	value, err := s.client.Get(ctx, stockKey(productID)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}

	stock, err := strconv.Atoi(value)
	if err != nil {
		return 0, false, err
	}
	return stock, true, nil
}

func (s *StockStore) Reserve(ctx context.Context, orderID string, items []StockItem, ttl time.Duration) error {
	if !s.Enabled() || len(items) == 0 {
		return nil
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	keys := make([]string, 0, len(items)+1)
	keys = append(keys, reservationKey(orderID))
	args := []interface{}{orderID, time.Now().Add(ttl).Unix()}
	for _, item := range items {
		keys = append(keys, stockKey(item.ProductID))
		args = append(args, item.ProductID, item.Qty)
	}

	result, err := reserveScript.Run(ctx, s.client, keys, args...).Slice()
	if err != nil {
		return err
	}
	code, detail := parseScriptResult(result)
	switch code {
	case reserveOK:
		return nil
	case reserveOutStock:
		return fmt.Errorf("%w: %s", ErrInsufficient, detail)
	case reserveMiss:
		return fmt.Errorf("%w: %s", ErrStockCacheMiss, detail)
	default:
		return fmt.Errorf("unexpected reserve stock result: %v", result)
	}
}

func (s *StockStore) Release(ctx context.Context, orderID string) error {
	if !s.Enabled() {
		return nil
	}
	_, err := releaseScript.Run(ctx, s.client, []string{reservationKey(orderID)}, orderID).Result()
	return err
}

func (s *StockStore) Confirm(ctx context.Context, orderID string) error {
	if !s.Enabled() {
		return nil
	}
	pipe := s.client.Pipeline()
	pipe.Del(ctx, reservationKey(orderID))
	pipe.ZRem(ctx, reservationExpirationsKey, orderID)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *StockStore) ExpiredReservations(ctx context.Context, now time.Time, limit int64) ([]string, error) {
	if !s.Enabled() {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}

	return s.client.ZRangeByScore(ctx, reservationExpirationsKey, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    strconv.FormatInt(now.Unix(), 10),
		Offset: 0,
		Count:  limit,
	}).Result()
}

func stockKey(productID string) string {
	return "stock:product:" + productID
}

func reservationKey(orderID string) string {
	return "reservation:order:" + orderID
}

func parseScriptResult(result []interface{}) (int64, string) {
	if len(result) == 0 {
		return 0, ""
	}

	code := toInt64(result[0])
	detail := ""
	if len(result) > 1 {
		detail = fmt.Sprint(result[1])
	}
	return code, detail
}

func toInt64(value interface{}) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	default:
		return 0
	}
}

var reserveScript = redis.NewScript(`
local reservation_key = KEYS[1]
local order_id = ARGV[1]
local expires_at = tonumber(ARGV[2])

for i = 2, #KEYS do
  local current = redis.call("GET", KEYS[i])
  if not current then
    return {-2, KEYS[i]}
  end

  local qty = tonumber(ARGV[(i - 2) * 2 + 4])
  if tonumber(current) < qty then
    return {-1, KEYS[i]}
  end
end

for i = 2, #KEYS do
  local product_id = ARGV[(i - 2) * 2 + 3]
  local qty = tonumber(ARGV[(i - 2) * 2 + 4])
  redis.call("DECRBY", KEYS[i], qty)
  redis.call("HSET", reservation_key, product_id, qty)
end

redis.call("ZADD", "reservation:order:expirations", expires_at, order_id)
return {1, ""}
`)

var rebuildScript = redis.NewScript(`
local snapshot = cjson.decode(ARGV[1])
for _, product_id in ipairs(redis.call("SMEMBERS", KEYS[1])) do
  redis.call("DEL", "stock:product:" .. product_id)
end
for _, order_id in ipairs(redis.call("ZRANGE", KEYS[2], 0, -1)) do
  redis.call("DEL", "reservation:order:" .. order_id)
end
redis.call("DEL", KEYS[1], KEYS[2])
for product_id, stock in pairs(snapshot.stocks) do
  redis.call("SET", "stock:product:" .. product_id, stock)
  redis.call("SADD", KEYS[1], product_id)
end
for _, reservation in ipairs(snapshot.reservations) do
  local key = "reservation:order:" .. reservation.order_id
  -- Also replace hashes whose expiration index was lost independently.
  redis.call("DEL", key)
  for _, item in ipairs(reservation.items) do
    redis.call("HSET", key, item.product_id, item.qty)
  end
  redis.call("ZADD", KEYS[2], reservation.expires_at, reservation.order_id)
end
return 1
`)

var releaseScript = redis.NewScript(`
local reservation_key = KEYS[1]
local items = redis.call("HGETALL", reservation_key)

if #items == 0 then
  return 0
end

for i = 1, #items, 2 do
  local product_id = items[i]
  local qty = tonumber(items[i + 1])
  redis.call("INCRBY", "stock:product:" .. product_id, qty)
end

redis.call("DEL", reservation_key)
redis.call("ZREM", "reservation:order:expirations", ARGV[1])
return 1
`)
