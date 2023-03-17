package view_default

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	. "github.com/infrago/base"
	"github.com/infrago/infra"
	"github.com/infrago/view"
)

type (
	defaultDriver  struct{}
	defaultConnect struct {
		config view.Config
	}

	defaultParser struct {
		connect  *defaultConnect
		viewbody view.Body

		engine *template.Template
		layout string
		path   string
		model  Map //layout用的model
		body   string

		title, author, description, keywords string
		metas, styles, scripts               []string
	}
)

// 连接
func (driver *defaultDriver) Connect(config view.Config) (view.Connect, error) {
	if config.Left == "" {
		config.Left = "{%"
	}
	if config.Right == "" {
		config.Right = "%}"
	}
	if config.Root == "" {
		config.Right = "asset/views"
	}
	if config.Shared == "" {
		config.Right = "shared"
	}
	return &defaultConnect{
		config: config,
	}, nil
}

// 打开连接
func (connect *defaultConnect) Open() error {
	return nil
}
func (connect *defaultConnect) Health() (view.Health, error) {
	// connect.mutex.RLock()
	// defer connect.mutex.RUnlock()
	return view.Health{Workload: 0}, nil
}

// 关闭连接
func (connect *defaultConnect) Close() error {
	return nil
}

// 解析接口
func (connect *defaultConnect) Parse(body view.Body) (string, error) {
	parser := connect.newDefaultViewParser(body)
	if body, err := parser.Parse(); err != nil {
		return "", err
	} else {
		return body, nil
	}

}

func (connect *defaultConnect) newDefaultViewParser(body view.Body) *defaultParser {
	config := connect.config

	parser := &defaultParser{
		connect: connect, viewbody: body,
	}

	parser.metas = []string{}
	parser.styles = []string{}
	parser.scripts = []string{}

	helpers := Map{}
	for k, v := range body.Helpers {
		helpers[k] = v
	}

	//系统自动的函数库，
	helpers["layout"] = parser.layoutHelper
	helpers["title"] = parser.titleHelper
	helpers["author"] = parser.authorHelper
	helpers["keywords"] = parser.keywordsHelper
	helpers["description"] = parser.descriptionHelper
	helpers["body"] = parser.bodyHelper
	helpers["render"] = parser.renderHelper
	helpers["meta"] = parser.metaHelper
	helpers["metas"] = parser.metasHelper
	helpers["style"] = parser.styleHelper
	helpers["styles"] = parser.stylesHelper
	helpers["script"] = parser.scriptHelper
	helpers["scripts"] = parser.scriptsHelper

	parser.engine = template.New("default").Delims(config.Left, config.Right).Funcs(helpers)

	return parser
}

func (parser *defaultParser) Parse() (string, error) {
	return parser.Layout()
}

func (parser *defaultParser) Layout() (string, error) {
	config := parser.connect.config
	body := parser.viewbody

	bodyText, bodyError := parser.Body(body.View, body.Model)
	if bodyError != nil {
		return "", bodyError
	}

	if parser.layout == "" {
		//没有使用布局，直接返回BODY
		return bodyText, nil
	}

	if parser.model == nil {
		parser.model = Map{}
	}

	//body赋值
	parser.body = bodyText

	var viewName, layoutHtml string
	if strings.Contains(parser.layout, "\n") {
		viewName = infra.Generate()
		layoutHtml = parser.layout
	} else {

		//先搜索layout所在目录
		viewpaths := []string{}

		if parser.path != "" {
			viewpaths = append(viewpaths, fmt.Sprintf("%s/%s.html", parser.path, parser.layout))
		}
		viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s.html", config.Root, body.Language, parser.layout))
		if body.Site != "" {
			viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s/%s.html", config.Root, body.Site, body.Language, parser.layout))
			viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s/%s/%s.html", config.Root, body.Site, body.Language, config.Shared, parser.layout))
		}
		viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s.html", config.Root, body.Language, parser.layout))

		if body.Site != "" {
			viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s/%s.html", config.Root, body.Site, config.Shared, parser.layout))
			viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s.html", config.Root, body.Site, parser.layout))
		}
		viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s.html", config.Root, config.Shared, parser.layout))
		viewpaths = append(viewpaths, fmt.Sprintf("%s/%s.html", config.Root, parser.layout))

		var filename string
		for _, s := range viewpaths {
			if f, _ := os.Stat(s); f != nil && !f.IsDir() {
				filename = s
				break
			}
		}
		//如果view不存在
		if filename == "" {
			return "", errors.New(fmt.Sprintf("layout %s not exist", parser.layout))
		}

		//读文件
		bytes, err := ioutil.ReadFile(filename)
		if err != nil {
			return "", errors.New(fmt.Sprintf("layout %s read error", parser.layout))
		}

		viewName = path.Base(filename)
		layoutHtml = string(bytes)
	}

	//不直接使用 parser.engine 来new,而是克隆一份
	engine, _ := parser.engine.Clone()
	t, e := engine.New(viewName).Parse(layoutHtml)
	if e != nil {
		return "", errors.New(fmt.Sprintf("layout %s parse error: %v", viewName, e))
	}

	//缓冲
	buf := bytes.NewBuffer(make([]byte, 0))

	//viewdata
	data := Map{}
	for k, v := range body.Data {
		data[k] = v
	}
	data["model"] = parser.model

	e = t.Execute(buf, data)
	if e != nil {
		return "", errors.New(fmt.Sprintf("layout %s parse error: %v", viewName, e))
	} else {
		return buf.String(), nil
	}
}

/* 返回view */
func (parser *defaultParser) Body(name string, args ...Any) (string, error) {
	config := parser.connect.config
	body := parser.viewbody

	var bodyModel Any
	if len(args) > 0 {
		bodyModel = args[0]
	}

	var viewName, bodyHtml string
	if strings.Contains(name, "\n") {
		viewName = infra.Generate()
		bodyHtml = name
	} else {

		viewpaths := []string{}
		if body.Site != "" {
			viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s/%s.html", config.Root, body.Site, body.Language, name))
			viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s/%s/%s.html", config.Root, body.Site, config.Shared, body.Language, name))
		}
		viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s.html", config.Root, body.Language, name))
		viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s/index.html", config.Root, body.Language, name))
		viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s/%s.html", config.Root, body.Language, config.Shared, name))
		viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s/%s/index.html", config.Root, body.Language, config.Shared, name))
		if body.Site != "" {
			viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s.html", config.Root, body.Site, name))
			viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s/index.html", config.Root, body.Site, name))
			viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s/%s.html", config.Root, body.Site, config.Shared, name))
		}
		viewpaths = append(viewpaths, fmt.Sprintf("%s/%s.html", config.Root, name))
		viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/index.html", config.Root, name))
		viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s.html", config.Root, config.Shared, name))
		viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s/index.html", config.Root, config.Shared, name))

		var filename string
		for _, s := range viewpaths {
			if f, _ := os.Stat(s); f != nil && !f.IsDir() {
				filename = s
				//这里要保存body所在的目录，为当前目录
				parser.path = filepath.Dir(s)
				break
			}
		}

		//如果view不存在
		if filename == "" {
			return "", errors.New(fmt.Sprintf("view %s not exist", name))
		}

		viewName = path.Base(filename)

		//读文件
		bytes, err := ioutil.ReadFile(filename)
		if err != nil {
			return "", errors.New(fmt.Sprintf("view %s read error", viewName))
		}

		bodyHtml = string(bytes)
	}

	//不直接使用 parser.engine 来new,而是克隆一份，这是为什么？
	engine, _ := parser.engine.Clone()
	t, e := engine.New(viewName).Parse(bodyHtml)
	if e != nil {
		return "", errors.New(fmt.Sprintf("view %s parse error: %v", viewName, e))
	}

	//缓冲
	buf := bytes.NewBuffer(make([]byte, 0))

	//viewdata
	data := Map{}
	for k, v := range body.Data {
		data[k] = v
	}
	data["model"] = bodyModel

	e = t.Execute(buf, data)
	if e != nil {
		return "", errors.New(fmt.Sprintf("view %s parse error: %v", viewName, e))
	} else {
		return buf.String(), nil
	}

}

/* 返回view */
func (parser *defaultParser) Render(name string, args ...Any) (string, error) {
	config := parser.connect.config
	body := parser.viewbody

	var renderModel Any
	if len(args) > 0 {
		renderModel = args[0]
	} else {
		renderModel = make(Map)
	}

	var viewName, renderHtml string
	if strings.Contains(name, "\n") {
		viewName = infra.Generate()
		renderHtml = name
	} else {

		//先搜索body所在目录
		viewpaths := []string{}
		if parser.path != "" {
			viewpaths = append(viewpaths, fmt.Sprintf("%s/%s.html", parser.path, name))
		}
		if body.Site != "" {
			viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s/%s/%s.html", config.Root, body.Site, body.Language, config.Shared, name))
			viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s/%s.html", config.Root, body.Site, body.Language, name))
		}
		viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s/%s.html", config.Root, body.Language, config.Shared, name))
		viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s.html", config.Root, body.Language, name))
		if body.Site != "" {
			viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s/%s.html", config.Root, body.Site, config.Shared, name))
			viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s.html", config.Root, body.Site, name))
		}
		viewpaths = append(viewpaths, fmt.Sprintf("%s/%s/%s.html", config.Root, config.Shared, name))
		viewpaths = append(viewpaths, fmt.Sprintf("%s/%s.html", config.Root, name))

		var filename string
		for _, s := range viewpaths {
			if f, _ := os.Stat(s); f != nil && !f.IsDir() {
				filename = s
				break
			}
		}

		//如果view不存在
		if filename == "" {
			return "", errors.New(fmt.Sprintf("render %s not exist", name))
		}

		//读文件
		bytes, err := ioutil.ReadFile(filename)
		if err != nil {
			return "", errors.New(fmt.Sprintf("layout %s read error", parser.layout))
		}

		viewName = path.Base(filename)
		renderHtml = string(bytes)

	}

	//不直接使用 parser.engine 来new,而是克隆一份
	//因为1.6以后，不知道为什么，直接用，就会有问题
	//会报重复render某页面的问题
	engine, _ := parser.engine.Clone()

	//如果一个模板被引用过了
	//不再重新加载文件
	//要不然, render某个页面,只能render一次
	t := engine.Lookup(viewName)

	if t == nil {
		newT, e := engine.New(viewName).Parse(renderHtml)
		if e != nil {
			return "", errors.New(fmt.Sprintf("render %s parse error: %v", viewName, e.Error()))
		} else {
			t = newT
		}
	}

	//缓冲
	buf := bytes.NewBuffer(make([]byte, 0))

	//viewdata
	data := Map{}
	for k, v := range body.Data {
		data[k] = v
	}
	data["model"] = renderModel

	e := t.Execute(buf, data)
	if e != nil {
		return "", errors.New(fmt.Sprintf("view %s parse error: %v", viewName, e))
	} else {
		return buf.String(), nil
	}

}

//--------------自带的helper

func (parser *defaultParser) layoutHelper(name string, vals ...Any) string {
	args := []Map{}
	for _, v := range vals {
		switch t := v.(type) {
		case Map:
			args = append(args, t)
		case string:
			m := Map{}
			e := infra.UnmarshalJSON([]byte(t), &m)
			if e == nil {
				args = append(args, m)
			}
		}
	}

	parser.layout = name
	if len(args) > 0 {
		parser.model = args[0]
	} else {
		parser.model = Map{}
	}

	return ""
}
func (parser *defaultParser) titleHelper(args ...string) template.HTML {
	if len(args) > 0 {
		//设置TITLE
		parser.title = args[0]
		return template.HTML("")
	} else {
		if parser.title != "" {
			return template.HTML(parser.title)
		} else {
			return template.HTML("")
		}
	}
}
func (parser *defaultParser) authorHelper(args ...string) template.HTML {
	if len(args) > 0 {
		//设置author
		parser.author = args[0]
		return template.HTML("")
	} else {
		if parser.author != "" {
			return template.HTML(parser.author)
		} else {
			return template.HTML("")
		}
	}
}
func (parser *defaultParser) keywordsHelper(args ...string) template.HTML {
	if len(args) > 0 {
		//设置TITLE
		parser.keywords = args[0]
		return template.HTML("")
	} else {
		if parser.keywords != "" {
			return template.HTML(parser.keywords)
		} else {
			return template.HTML("")
		}
	}
}
func (parser *defaultParser) descriptionHelper(args ...string) template.HTML {
	if len(args) > 0 {
		//设置TITLE
		parser.description = args[0]
		return template.HTML("")
	} else {
		if parser.description != "" {
			return template.HTML(parser.description)
		} else {
			return template.HTML("")
		}
	}
}
func (parser *defaultParser) bodyHelper() template.HTML {
	return template.HTML(parser.body)
}

func (parser *defaultParser) renderHelper(name string, vals ...Any) template.HTML {
	// args := []Map{}
	// for _, v := range vals {
	// 	if t, ok := v.(string); ok {
	// 		m := Map{}
	// 		e := infra.Unmarshal([]byte(t), &m)
	// 		if e == nil {
	// 			args = append(args, m)
	// 		}
	// 	} else if t, ok := v.(Map); ok {
	// 		args = append(args, t)
	// 	} else if ts, ok := v.([]Map); ok {
	// 		args = append(args, ts...)
	// 	} else {

	// 	}
	// }

	s, e := parser.Render(name, vals...)
	if e == nil {
		return template.HTML(s)
	} else {
		return template.HTML(fmt.Sprintf("render error: %v", e))
	}
}

func (parser *defaultParser) metaHelper(name, content string, https ...bool) string {
	isHttp := false
	if len(https) > 0 {
		isHttp = https[0]
	}
	if isHttp {
		parser.metas = append(parser.metas, fmt.Sprintf(`<meta http-equiv="%v" content="%v" />`, name, content))
	} else {
		parser.metas = append(parser.metas, fmt.Sprintf(`<meta name="%v" content="%v" />`, name, content))
	}
	return ""
}

func (parser *defaultParser) metasHelper() template.HTML {
	html := ""
	if len(parser.metas) > 0 {
		html = strings.Join(parser.metas, "\n")
	}
	return template.HTML(html)
}

func (parser *defaultParser) styleHelper(path string, args ...string) string {
	media := ""
	if len(args) > 0 {
		media = args[0]
	}
	if media == "" {
		parser.styles = append(parser.styles, fmt.Sprintf(`<link type="text/css" rel="stylesheet" href="%v" />`, path))
	} else {
		parser.styles = append(parser.styles, fmt.Sprintf(`<link type="text/css" rel="stylesheet" href="%v" media="%v" />`, path, media))
	}

	return ""
}

func (parser *defaultParser) stylesHelper() template.HTML {
	html := ""
	if len(parser.styles) > 0 {
		html = strings.Join(parser.styles, "\n")
	}
	return template.HTML(html)
}

func (parser *defaultParser) scriptHelper(path string, args ...string) string {
	tttt := "text/javascript"
	if len(args) > 0 {
		tttt = args[0]
	}
	parser.scripts = append(parser.scripts, fmt.Sprintf(`<script type="%v" src="%v"></script>`, tttt, path))

	return ""
}

func (parser *defaultParser) scriptsHelper() template.HTML {
	html := ""
	if len(parser.scripts) > 0 {
		html = strings.Join(parser.scripts, "\n")
	}

	return template.HTML(html)
}
