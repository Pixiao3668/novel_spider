package model

import "strconv"

// 配置结构体
type Config struct {
	RootDir string
	DelTempDir bool
	TermBarWidth int
	Timeout int
	KeyWords []string
}

var keywords = []string{
	"换源App",
	"huanyuanapp.org",
}

func DefaultConfig() *Config {
	return &Config{
		RootDir: "./novels",
		DelTempDir: true,
		TermBarWidth: 50,
		Timeout: 20,
		KeyWords: keywords,
	}
}

func (c *Config) ToString() string {
	return "\n{ \n RootDir(小说存放目录): " + c.RootDir + 
	"\n DelTempDir(使用删除临时目录): " + strconv.FormatBool(c.DelTempDir) + 
	"\n TermBarWidth(下载进度条宽度): " + strconv.Itoa(c.TermBarWidth) + 
	"\n Timeout(请求超时时间/秒): " + strconv.Itoa(c.Timeout) + 
	"\n}"
}