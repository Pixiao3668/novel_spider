package init

import (
	"fmt"
	"spider/internal/model"

	"github.com/spf13/viper"
)

var config *model.Config

func InitConfig() *model.Config {
	viper.SetConfigName("config") // 配置文件名称(无扩展名)
	viper.SetConfigType("yaml") // 配置文件类型
	viper.AddConfigPath("./configs")   // 配置文件所在文件夹
	viper.AddConfigPath(".")           // 额外的可以找到配置文件的路径
	err := viper.ReadInConfig() // 查找并读取配置文件
	if err != nil { // 处理读取配置文件的错误
		handleConfigError("读取配置文件失败")
		return config
	}
	// 将配置文件中的配置项映射到结构体中
	err = viper.UnmarshalKey("novel", &config)
	if err != nil {
		handleConfigError("配置文件映射失败")
		return config
	}
	// 将配置文件中的广告过滤关键词映射到结构体中
	config.KeyWords = viper.GetStringSlice("filter.keywords")
	return config
}

func handleConfigError(err string) {
	fmt.Printf("配置文件映射失败\n")
	config = model.DefaultConfig()
	fmt.Printf("已使用默认配置：%s \n", config.ToString())
}