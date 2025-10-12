package common

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func DmUser(s *discordgo.Session, userID, content string, embed *discordgo.MessageEmbed) {
	dmChannel, e := s.UserChannelCreate(userID)
	if e != nil {
		fmt.Printf("Error creating DM channel for user %s: %v\n", userID, e)
		return
	}
	_, e = s.ChannelMessageSendComplex(dmChannel.ID, &discordgo.MessageSend{
		Content: content,
		Embed:   embed,
	})
	if e != nil {
		fmt.Println("Error sending DM:", e)
		return
	}
}