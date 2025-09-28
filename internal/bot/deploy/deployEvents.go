package deploy

import (
	"log"

	"github.com/Sush1sui/sniper_bot/internal/bot/events"
	"github.com/bwmarrin/discordgo"
)

var EventHandlers = []any{
	events.OnSnipeMudae,
}

func DeployEvents(sess *discordgo.Session) {
	for _, handler := range EventHandlers {
		sess.AddHandler(handler)
	}
	log.Println("Event handlers deployed successfully.")
}