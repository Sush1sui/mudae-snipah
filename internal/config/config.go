package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds the application's configuration
type Config struct {
	DiscordToken 	string
	ServerPort   	string
	AppID        	string
	ServerURL      	string
	VIPs           	[]string
	MoodengRoleID  	string
}

var GlobalConfig Config

// LoadConfig initializes the configuration with default values
func LoadConfig() (error) {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file")
	}
	GlobalConfig = Config{
		DiscordToken: os.Getenv("DISCORD_TOKEN"),
		ServerPort:   os.Getenv("SERVER_PORT"),
		AppID:        os.Getenv("APP_ID"),
		ServerURL:    os.Getenv("SERVER_URL"),
		VIPs:         strings.Split(os.Getenv("SNIPER_VIP_USERS"), ","),
		MoodengRoleID: os.Getenv("SNIPER_ROLE_ID"),
	}
	return nil
}