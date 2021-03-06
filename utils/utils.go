package utils

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/bitcav/nitr/version"
	"github.com/gofiber/fiber"
	"github.com/gofiber/logger"
	"github.com/spf13/viper"
)

func ConfigFileSetup() {
	if _, err := os.Stat("config.ini"); err != nil {
		configFile, err := os.OpenFile("config.ini", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		LogError(err)
		defer configFile.Close()

		defaultConfigOpts := []string{
			"port: 8000",
			"open_browser_on_startup: true",
			"save_logs: false",
			"ssl_enabled: false",
			"# ssl_certificate: /path/to/file.crt ",
			"# ssl_certificate_key: /path/to/file.key",
		}

		defaultConfig := strings.Join(defaultConfigOpts, "\n")
		_, err = configFile.WriteString(defaultConfig)
		LogError(err)
	}

	runPath, err := os.Getwd()
	LogError(err)

	viper.SetConfigName("config.ini")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(runPath)
	err = viper.ReadInConfig()
	if err != nil {
		LogError(err)
	}
}

//OpenBrowser opens default web browser in specific domain
func OpenBrowser(domain, port string) {
	url := domain + ":" + port
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

const charset = "abcdefghijkmnpqrstuvwxyz" +
	"123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

//RandString returns random string with specific length
func RandString(length int) string {
	return stringWithCharset(length, charset)
}

//StartMessage displays message on server start up
func StartMessage(protocol, port string) {
	fmt.Printf(`       
     _____________
    /            /\          _  __    
   /   /    /   / /   ___   (_)/ /_ ____
  /   /    /   / /   / _ \ / // __// __/    
 /            / /   /_//_//_/ \__//_/
/____________/ / 	    
\____________\/     v%v

Go to admin panel at %v://localhost:%v

`, version.Version, protocol, port)
}

func Logs(app *fiber.App) {
	saveLogs := viper.GetBool("save_logs")
	if saveLogs {
		logFile, err := os.OpenFile("nitr.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		//defer logFile.Close()

		log.SetOutput(logFile)

		cfg := logger.Config{
			Output:     logFile,
			TimeFormat: "2006/01/02 15:04:05",
			Format:     "${time} - ${method} ${path} - ${ip}\n",
		}

		app.Use(logger.New(cfg))
	}

}

func LogError(e error) {
	if e != nil {
		log.Println(e)
	}
}

func GetLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return fmt.Sprint(localAddr.IP)
}

func GetLocalPort() string {
	port := viper.GetString("port")
	if port == "" {
		port = "8000"
	}
	return port
}

func StartServer(app *fiber.App) {
	port := GetLocalPort()
	sslEnabled := viper.GetBool("ssl_enabled")
	if sslEnabled {
		cert := viper.GetString("ssl_certificate")
		key := viper.GetString("ssl_certificate_key")

		cer, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			log.Println("Invalid ssl certificate")
			LogError(err)
		}

		config := &tls.Config{Certificates: []tls.Certificate{cer}}
		StartMessage("https", port)

		openBrowser := viper.GetBool("open_browser_on_startup")
		if openBrowser {
			OpenBrowser("https://localhost", port)
		}

		log.Println("Starting server")

		err = app.Listen(port, config)
		if err != nil {
			fmt.Println(err, "\nCheck settings at config.ini file")
		}
		LogError(err)

	} else {
		StartMessage("http", port)
		openBrowser := viper.GetBool("open_browser_on_startup")
		if openBrowser {
			OpenBrowser("http://localhost", port)
		}

		log.Println("Starting server")

		err := app.Listen(port)
		if err != nil {
			fmt.Println(err, "\nCheck settings at config.ini file")
		}
		LogError(err)
	}
}

func PasswordHash(password string) string {
	h := sha256.New()
	h.Write([]byte(password))
	sha1_hash := hex.EncodeToString(h.Sum(nil))

	return sha1_hash
}
