package model

import "time"

type Account struct {
	UserID            string    `gorm:"column:userId;type:uuid;not null"`
	Type              string    `gorm:"column:type;primaryKey;type:text"`
	Provider          string    `gorm:"column:provider;primaryKey;type:text"`
	ProviderAccountID string    `gorm:"column:providerAccountId;primaryKey;type:text"`
	RefreshToken      *string   `gorm:"column:refresh_token;type:text"`
	AccessToken       *string   `gorm:"column:access_token;type:text"`
	ExpiresAt         *int      `gorm:"column:expires_at"`
	TokenType         *string   `gorm:"column:token_type;type:text"`
	Scope             *string   `gorm:"column:scope;type:text"`
	IDToken           *string   `gorm:"column:id_token;type:text"`
	SessionState      *string   `gorm:"column:session_state;type:text"`
	CreatedAt         time.Time `gorm:"column:createdAt;autoCreateTime"`
	UpdatedAt         time.Time `gorm:"column:updatedAt;autoUpdateTime"`

	User User `gorm:"foreignKey:UserID;references:ID"`
}

func (Account) TableName() string {
	return "Account"
}

type Session struct {
	SessionToken string    `gorm:"column:sessionToken;primaryKey;type:text"`
	UserID       string    `gorm:"column:userId;type:uuid;index;not null"`
	Expires      time.Time `gorm:"column:expires;not null"`
	CreatedAt    time.Time `gorm:"column:createdAt;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updatedAt;autoUpdateTime"`

	User User `gorm:"foreignKey:UserID;references:ID"`
}

func (Session) TableName() string {
	return "Session"
}

type VerificationToken struct {
	Identifier string    `gorm:"column:identifier;primaryKey;type:text"`
	Token      string    `gorm:"column:token;primaryKey;type:text"`
	Expires    time.Time `gorm:"column:expires;not null"`
}

func (VerificationToken) TableName() string {
	return "VerificationToken"
}
