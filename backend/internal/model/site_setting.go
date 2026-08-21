package model

// SiteSetting 站点设置（键值对）。
type SiteSetting struct {
	SettingKey   string `gorm:"column:setting_key;primaryKey;size:50" json:"key"`
	SettingValue string `gorm:"type:text;column:setting_value" json:"value"`
}
