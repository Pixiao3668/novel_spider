package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

var client *http.Client
var wg sync.WaitGroup
var url = flag.String("url", "", "小说目录页地址")

func init()  {
	// 创建一个 HTTP 客户端
	client = &http.Client{}
	wg = sync.WaitGroup{}
}

// 创建一个章节结构体
type Chapter struct {
	Index int
	Title string
	Href string
}

func main() {
	flag.Parse()
	if *url == "" {
		fmt.Println("没有输入小说目录页地址")
		return
	}
	baseUrl := strings.TrimRight(*url, "/")

	// 创建一个 HTTP 请求
	req, err := http.NewRequest("GET", baseUrl, nil)
	if err != nil {
		fmt.Printf("创建请求出错：%s\n",err);
		return
	}
	// User-Agent 设置成苹果浏览器
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko)")

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
	chapters := make([]Chapter, 0)
	novelName := doc.Find("#info > h1").Text()
	doc.Find("#list > dl > dd").Each(func(i int, s *goquery.Selection) {
		// 获取 a 标签的文本内容 和 herf 属性值
		title := s.Find("a").Text()
		href, _ := s.Find("a").Attr("href")
		// 将章节信息添加到切片中
		chapters = append(chapters, Chapter{
			Index: i+1,
			Title: title,
			Href: baseUrl + href,
		})
		fmt.Printf("第 %d 章节的标题：%s，链接：%s\n", i+1, title, baseUrl + href)
	})
	// 创建以小说名字为名的文件
	err = os.MkdirAll(novelName, 0777)
	if err != nil {
		fmt.Printf("创建《%s》文件夹出错：%s\n", novelName,err);
		return
	}
	wg.Add(len(chapters))
	
	// 使用通道统计已经下载的章节数
	countCh := make(chan string)
	defer close(countCh)
	go func() {
		for i := 0; i < len(chapters); i++ {
			chapName := <-countCh
			fmt.Printf("第 %d 章节 %s 已成功下载\n", i + 1,chapName)
		}
	}()
	// 遍历章节切片，获取每个章节的内容
	for _, chap := range chapters {
		go getChapterContent(chap, "./" + novelName, countCh)
	}
	wg.Wait()
}; 

func getChapterContent(chap Chapter, dir string, countCh chan string)  {
	// 创建一个 HTTP 请求
	req, err := http.NewRequest("GET", chap.Href, nil)
	if err != nil {
		fmt.Printf("创建请求出错：%s\n",err);
		return
	}
	// User-Agent 设置成苹果浏览器
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko)")

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

	file, err := os.OpenFile(dir + "/"+ strconv.Itoa(chap.Index) + ".txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		fmt.Printf("打开文件出错：%s\n",err);
		return
	}
	file.WriteString(chap.Title + "\n")
	defer file.Close()

	// 获取章节内容
	doc.Find("#content > p").Each(func(i int, s *goquery.Selection) {
		file.WriteString(s.Text() + "\n")
	})
	countCh <- chap.Title
	wg.Done()
}