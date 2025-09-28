package deploy

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

var EventHandlers = []any{
	
}

func DeployEvents(sess *discordgo.Session) {
	for _, handler := range EventHandlers {
		sess.AddHandler(handler)
	}
	log.Println("Event handlers deployed successfully.")
}