package models

type PushSubscription struct {
	BaseModel
	UserID   string `gorm:"size:255;index" json:"user_id"`
	Endpoint string `gorm:"size:500;uniqueIndex" json:"endpoint"`
	P256dh   string `gorm:"size:255" json:"p256dh"`
	Auth     string `gorm:"size:255" json:"auth"`
	
	User     *Usuario `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
