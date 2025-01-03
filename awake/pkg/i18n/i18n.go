package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var bundle *i18n.Bundle
var localizer *i18n.Localizer

// Init 初始化 i18n 包，需要传入语言文件所在的目录路径
func Init(langDir string) error {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// 确保加载了正确的语言文件
	entries, err := os.ReadDir(langDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			_, err := bundle.LoadMessageFile(filepath.Join(langDir, entry.Name()))
			if err != nil {
				return err
			}
		}
	}

	// 确保设置了正确的语言
	userLang := "zh-CN" // 强制使用中文
	localizer = i18n.NewLocalizer(bundle, userLang)
	return nil
}

// T 翻译指定的消息ID
func T(messageID string, args ...interface{}) string {
	if localizer == nil {
		return messageID
	}
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: args,
	})
	if err != nil {
		return messageID
	}
	return msg
}

// SetLanguage 设置当前语言
func SetLanguage(lang string) {
	if bundle != nil {
		localizer = i18n.NewLocalizer(bundle, lang)
	}
}
