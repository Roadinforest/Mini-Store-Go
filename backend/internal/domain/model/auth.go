package model

import "time"

type Account struct {
	UserID            string    `gorm:"column:user_id;primaryKey;type:uuid"`
	Type              string    `gorm:"column:type;primaryKey;type:text"`
	Provider          string    `gorm:"column:provider;primaryKey;type:text"`
	ProviderAccountID string    `gorm:"column:provider_account_id;primaryKey;type:text"`
	RefreshToken      *string   `gorm:"column:refresh_token;type:text"`
	AccessToken       *string   `gorm:"column:access_token;type:text"`
	ExpiresAt         *int      `gorm:"column:expires_at"`
	TokenType         *string   `gorm:"column:token_type;type:text"`
	Scope             *string   `gorm:"column:scope;type:text"`
	IDToken           *string   `gorm:"column:id_token;type:text"`
	SessionState      *string   `gorm:"column:session_state;type:text"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime"`

	User User `gorm:"foreignKey:UserID;references:ID"`
}

func (Account) TableName() string {
	return "accounts"
}

type Session struct {
	SessionToken string    `gorm:"column:session_token;primaryKey;type:text"`
	UserID       string    `gorm:"column:user_id;type:uuid;index;not null"`
	Expires      time.Time `gorm:"column:expires;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`

	User User `gorm:"foreignKey:UserID;references:ID"`
}

func (Session) TableName() string {
	return "sessions"
}

type VerificationToken struct {
	Identifier string    `gorm:"column:identifier;primaryKey;type:text"`
	Token      string    `gorm:"column:token;primaryKey;type:text"`
	Expires    time.Time `gorm:"column:expires;not null"`
}

func (VerificationToken) TableName() string {
	return "verification_tokens"
}
