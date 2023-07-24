package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	ini "spider/init"
	"spider/internal/model"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/schollz/progressbar/v3"
)

var client *http.Client
var wg sync.WaitGroup
var url = flag.String("url", "", "小说目录页地址")
var config *model.Config

func init()  {
	wg = sync.WaitGroup{}
	// 配置初始化
	config = ini.InitConfig()
	// 创建一个 HTTP 客户端
	client = &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}
}

func main() {
	flag.Parse()
	if *url == "" {
		fmt.Println("没有输入小说目录页地址")
		return
	}
	baseUrl := strings.TrimRight(*url, "/")
	getChapterList(baseUrl)
}; 

// 获取章节列表
func getChapterList(baseUrl string) {
	// 创建一个 HTTP 请求
	req, err := http.NewRequest("GET", baseUrl, nil)
	if err != nil {
		fmt.Printf("创建请求出错：%s\n",err);
		return
	}
	// User-Agent 设置成苹果浏览器
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/66.0.3359.181 Safari/537.36")

	// 发起请求
	res, err := client.Do(req)
	if err != nil {
		fmt.Printf("请求出错：%s\n",err);
		return
	}
	defer res.Body.Close();

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		fmt.Printf("解析出错：%s\n",err);
		return
	}
	// 创建一个章节切片
	chapters := make([]model.Chapter, 0)
	novelName := doc.Find("#info > h1").Text()
	// 创建以小说名字为名的文件
	novelPath := path.Join(config.RootDir, novelName)
	err = os.MkdirAll(novelPath, 0777)
	if err != nil {
		fmt.Printf("创建文件夹出错：%s\n",err);
		return
	}
	doc.Find("#list > dl > dd").Each(func(i int, s *goquery.Selection) {
		// 获取 a 标签的文本内容 和 herf 属性值
		title := s.Find("a").Text()
		href, _ := s.Find("a").Attr("href")
		// 将章节信息添加到切片中
		chapters = append(chapters, model.Chapter{
			Index: i+1,
			Title: title,
			Href: baseUrl + href,
			Path: path.Join(novelPath, strconv.Itoa(i+1) + ".txt"),
		})
	})

	wg.Add(len(chapters))

	// 创建一个进度条
	bar := progressbar.NewOptions(len(chapters),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetWidth(config.TermBarWidth),
		progressbar.OptionSetDescription("【[red]" + novelName + "[reset]】" + "章节下载中 ..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "🐌",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	defer bar.Close()
	// 使用通道统计已经下载的章节数
	countCh := make(chan string)
	defer close(countCh)
	go func() {
		for i := 0; i < len(chapters); i++ {
			<- countCh
			// fmt.Printf("第 %d 章节 %s 已成功下载\n", i + 1,chapName)
			bar.Add(1)
		}
	}()
	// 遍历章节切片，获取每个章节的内容
	for _, chap := range chapters {
		go getChapterContent(chap, novelPath, countCh)
	}
	wg.Wait()
	// 将临时文件夹中的文件合成一个txt文件
	mergeNovel(&chapters, novelName)

	// 删除临时文件夹
	checkNovelTemp(novelPath)
}

// 获取每个章节的内容
func getChapterContent(chap model.Chapter, dir string, countCh chan string)  {
	// 创建一个 HTTP 请求
	req, err := http.NewRequest("GET", chap.Href, nil)
	if err != nil {
		fmt.Printf("创建请求出错：%s\n",err);
		return
	}
	// User-Agent 设置成苹果浏览器
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/66.0.3359.181 Safari/537.36")


	// 发起请求
	res, err := client.Do(req)
	if err != nil {
		fmt.Printf("请求出错：%s\n",err);
		return
	}
	defer res.Body.Close();

	if res.StatusCode != 200 {
		fmt.Printf("请求出错：%s\n",err);
		return
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		fmt.Printf("解析出错：%s\n",err);
		return
	}

	file, err := os.OpenFile(chap.Path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		fmt.Printf("打开文件出错：%s\n",err);
		return
	}
	file.WriteString(chap.Title + "\n")
	defer file.Close()

	// 获取章节内容
	doc.Find("#content > p").Each(func(i int, s *goquery.Selection) {
		ptext := s.Text()
		// 广告判断
		res := containsKeyword(ptext)
		if(!res) {
			file.WriteString(ptext + "\n")
		}
		
	})
	countCh <- chap.Title
	wg.Done()
}

// 合并小说
func mergeNovel(chapters *[]model.Chapter, novelName string)  {
	novelPath := path.Join(config.RootDir, novelName + ".txt")
	// 判断文件是否存在，存在就删除
	if _, err := os.Stat(novelPath); err == nil {
		err := os.Remove(novelPath)
		if err != nil {
			panic(fmt.Errorf("删除文件失败: %w", err))
		}
	}
	// 创建一个文件
	file, err := os.OpenFile(novelPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		fmt.Printf("【%s】创建失败：%s\n", novelName + ".txt",err);
		return
	}
	defer file.Close()
	for _, chap := range *chapters {
		// 打开章节文件
		chapFile, err := os.OpenFile(chap.Path, os.O_RDWR, 0777)
		if err != nil {
			fmt.Printf("【%s】打开失败：%s\n", chap.Path,err);
			return
		}
		defer chapFile.Close()
		content, err := os.ReadFile(chap.Path)
		if err != nil {
			fmt.Printf("【%s】读取失败：%s\n", chap.Path,err);
			return
		}
		file.WriteString(string(content))
	}
}

// 检查小说临时文件夹
func checkNovelTemp(novelPath string) {
	if(config.DelTempDir) {
		err := os.RemoveAll(novelPath)
		if err != nil {
			panic(fmt.Errorf("删除临时文件夹失败: %w", err))
		}
	}
}

func containsKeyword(str string) bool {
	for _, keyword := range config.KeyWords {
		if strings.Contains(str, keyword) {
			return true
		}
	}
	return false
}