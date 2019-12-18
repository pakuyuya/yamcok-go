package main

import (
    "github.com/labstack/echo"
    "github.com/labstack/echo/middleware"
    shellwords "github.com/mattn/go-shellwords"
    "fmt"
    "log"
	"os"
	"os/exec"
	"net/http"
	"io/ioutil"
	"time"
	"strconv"
	"strings"
	r "reflect"
    yaml "gopkg.in/yaml.v2"
)

const mockfileroot = "./json"

type Settings struct {
	Port string `yaml:"port"`
	Paths []map[string]interface{} `yaml:"paths"`
}


var settings Settings

func main() {

    filePath := "setting.yaml"
	if len(os.Args) > 1 {
		filePath = os.Args[1]
	}

    data, _ := ioutil.ReadFile(filePath)
	
	yaml.Unmarshal(data, &settings)

	e := echo.New()
	
	e.Use(middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
		os.Stdout.Write([]byte("----------------------------------------------------\n"))
		os.Stdout.Write([]byte(time.Now().String()))
		os.Stdout.Write([]byte("\n"))
		os.Stdout.Write([]byte("--- request ---\n"))
		output := fmt.Sprintf("[" + c.Request().Method + "] " + c.Request().URL.Path+"\n[headers] = %#v \n", c.Request().Header)
		os.Stdout.Write([]byte(output))
		os.Stdout.Write([]byte("[body] = " + string(reqBody) + "\n"))
		os.Stdout.Write([]byte("--- response ---\n"))
		os.Stdout.Write([]byte("StatusCode: " + strconv.Itoa(c.Response().Status) + " Body: " + string(resBody) + "\n"))
		os.Stdout.Write([]byte(time.Now().String()))
		os.Stdout.Write([]byte("\n"))
		os.Stdout.Write([]byte("----------------------------------------------------\n"))
	}))
	e.Any("/*", handler)

	port := "8888"
	if settings.Port != "" {
		port = settings.Port
	}

    log.Fatal(e.Start(":" + port))
}

func handler(c echo.Context) error {
	path := c.Request().URL.Path
	method := strings.ToLower(c.Request().Method)

	pp := findPath(path)

	// not found
	if pp == nil {
		return echo.NewHTTPError(http.StatusNotFound, `404: content not found`)
	}
	
	p := *pp

	if p["methods"] != nil {
		allowMethods := strings.Split(p["methods"].(string), ",")
		
		notfound := true
		for _, m := range allowMethods {
			lm := strings.ToLower(strings.TrimSpace(m))
			if lm == method {
				notfound = false
				break
			}
		}
		if notfound {
			return echo.NewHTTPError(http.StatusNotFound, `404: content not found`)
		}
	}

	if p["command"] != nil {
		c, _ := shellwords.Parse(p["command"].(string))
		cmd := exec.Command(c[0])
		if len(c) > 1 {
			cmd = exec.Command(c[0], c[1:]...)
		}
		cmd.Run()
	}
	
	status := 200
	body := make([]byte, 0)

	if p["response"] != nil {
		pres := p["response"].(map[interface{}]interface{})

		if pres["status"] != nil {
			status = pres["status"].(int)
		}
	
		if pres["headers"] != nil {
			headers := pres["headers"].(map[interface{}]interface{})
			for header, vHeader := range headers {
				c.Response().Header().Set(header.(string), vHeader.(string))
			}
		}
		if pres["bodyfile"] != nil {
			body, _ = ioutil.ReadFile(pres["bodyfile"].(string))
		}
	}
	return c.String(status, string(body))
}

func findPath(path string) *map[string]interface{} {
	for _, p := range settings.Paths {
		if p["path"] != nil {
			stgpath := p["path"].(string)
			if !strings.HasPrefix(stgpath, "/") {
				stgpath = "/" + stgpath 
			}
			if stgpath == path {
				return &p
			}
		}
	}
	return nil
}

func mapToStruct(mapVal map[string]interface{}, val interface{}) (ok bool) {
	structVal:= r.Indirect(r.ValueOf(val))
	for name, elem := range mapVal {
		structVal.FieldByName(name).Set(r.ValueOf(elem))
	}

	return
}

