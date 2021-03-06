package main

import (
	"flag"
	"fmt"
	"github.com/disintegration/gift"
	_ "github.com/k0kubun/pp"
	"image"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

var portNumber = flag.String("port", "8080", "port number.")
var basicAuthUser = flag.String("user", "", "basic auth user name")
var basicAuthPass = flag.String("pass", "", "basic auth user pass")

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello")
}

func printMultiPage(w http.ResponseWriter, url1 string, url2 string) {
	fmt.Fprintf(w, `<html><head><title>luigi</title></head>
<body>
<script>
window.open('%s', 'url1', 'width=600, height=600, menubar=no, toolbar=no, scrollbars=yes');
window.open('%s', 'url2', 'width=600, height=600, menubar=no, toolbar=no, scrollbars=yes');
</script>
</body>
</html>
`, url1, url2)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// pp.Print(r)

	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	imagedir := path.Join(dir, "images")
	if err := os.Mkdir(imagedir, 0755); err != nil && !os.IsExist(err) {
		fmt.Fprintln(w, err)
		return
	}
	file, _, err := r.FormFile("imagedata")
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	defer file.Close()
	basename := strconv.FormatInt(time.Now().UnixNano(), 10) + ".png"
	imagefile := path.Join(imagedir, basename)
	out, err := os.Create(imagefile)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	log.Println(r.Host)

	// pp.Print(header)
	// fmt.Fprintf(w, "http://%s/images/%s", r.Host, basename)
	if r.URL.RawQuery == "cannotmulti" {
		url := fmt.Sprintf("http://%s/similar/%s", r.Host, basename)
		fmt.Fprintf(w, "%s", url)
	} else {
		url1 := fmt.Sprintf("http://%s/similar1/%s,", r.Host, basename)
		url2 := fmt.Sprintf("http://%s/similar2/%s,", r.Host, basename)
		fmt.Fprintf(w, "%s,%s", url1, url2)
	}
}

func checkAuth(w http.ResponseWriter, r *http.Request) bool {
	if *basicAuthUser == "" || *basicAuthPass == "" {
		return true
	}

	username, password, ok := r.BasicAuth()
	// log.Println(username, password, ok)
	if ok == false {
		return false
	}
	return username == *basicAuthUser && password == *basicAuthPass
}

func imageURL(r *http.Request, replace string, replaceTo string) string {
	return fmt.Sprintf("http://%s%s", r.Host, strings.Replace(r.URL.Path, replace, replaceTo, 1))
}

func imageSearchURL(url string) string {
	return fmt.Sprintf(`https://www.google.co.jp/searchbyimage?image_content=&filename=&safe=off&hl=ja&authuser=0&image_url=%s`, url)
}

func imagePath(r *http.Request, replace string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		log.Println(err)
		return "", err
	}
	return path.Join(dir, strings.Replace(r.URL.Path, replace, "images", 1)), nil
}

func imagesHandler(w http.ResponseWriter, r *http.Request) {
	if checkAuth(w, r) == false {
		w.Header().Set("WWW-Authenticate", `Basic realm="Atto"`)
		w.WriteHeader(401)
		w.Write([]byte("401 Unauthorized\n"))
		return
	}

	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	// pp.Print(r)
	imagefile := path.Join(dir, r.URL.Path)
	log.Println(imagefile)
	http.ServeFile(w, r, imagefile)
}

func similarHandler(w http.ResponseWriter, r *http.Request) {
	url1 := imageURL(r, "similar", "similar1")
	url2 := imageURL(r, "similar", "similar2")
	log.Println(url1)
	printMultiPage(w, url1, url2)
}

func similar1Handler(w http.ResponseWriter, r *http.Request) {
	url := imageSearchURL(imageURL(r, "similar1", "images"))
	log.Println(url)
	http.Redirect(w, r, url, 302)
}

func similar2Handler(w http.ResponseWriter, r *http.Request) {
	img, err := imagePath(r, "similar2")
	if err != nil {
		log.Println("get image path", err)
		return
	}
	in, err := os.Open(img)
	if err != nil {
		log.Println("error open file", err)
		return
	}
	defer in.Close()

	src, err := png.Decode(in)
	if err != nil {
		log.Println("error open file", err)
		return
	}

	g := gift.New(gift.FlipHorizontal())
	dst := image.NewRGBA(g.Bounds(src.Bounds()))
	g.Draw(dst, src)

	outFile := strings.Replace(img, ".png", "_flipH.png", 1)

	out, err := os.OpenFile(outFile, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Println("error open file", err)
		return
	}
	defer out.Close()

	err = png.Encode(out, dst)
	if err != nil {
		log.Println("error encode file", err)
		return
	}

	url := imageSearchURL(imageURL(r, "similar2", "images"))
	log.Println(url)
	url2 := strings.Replace(url, ".png", "_flipH.png", 1)
	log.Println(url2)

	http.Redirect(w, r, url2, 302)
}

func main() {
	flag.Parse()
	if *basicAuthUser != "" && *basicAuthPass != "" {
		log.Println("basic auth: " + *basicAuthUser)
	}
	log.Println("listen:" + *portNumber)

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/images/", imagesHandler)
	http.HandleFunc("/similar/", similarHandler)
	http.HandleFunc("/similar1/", similar1Handler)
	http.HandleFunc("/similar2/", similar2Handler)
	http.HandleFunc("/upload", uploadHandler)
	log.Fatal(http.ListenAndServe(":"+*portNumber, nil))
}
