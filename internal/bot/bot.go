package bot

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sush1sui/sniper_bot/internal/bot/deploy"
	"github.com/Sush1sui/sniper_bot/internal/config"
	"github.com/bwmarrin/discordgo"
)

func StartBot() {
	// create new discord session
	if config.GlobalConfig.DiscordToken == "" {
		fmt.Println("Bot token not found")
	}
	sess, err := discordgo.New("Bot " + config.GlobalConfig.DiscordToken)
	if err != nil {
		log.Fatalf("error creating Discord session: %v", err)
	}

	sess.Identify.Intents = discordgo.IntentsAllWithoutPrivileged | discordgo.IntentsGuildPresences | discordgo.IntentsGuildMembers | discordgo.IntentsGuildMessages

	sess.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
    s.UpdateStatusComplex(discordgo.UpdateStatusData{
        Status: "idle",
        Activities: []*discordgo.Activity{
            {
                Name: "Mudae!",
                Type: discordgo.ActivityTypeListening,
            },
        },
    })
	})

	err = sess.Open()
	if err != nil {
		log.Fatalf("error opening connection to Discord: %v", err)
	}
	defer sess.Close()

	// Check read message history permission for MUDAE channel
    mudaeChannel := os.Getenv("MUDAE_CHANNEL_ID")
    if mudaeChannel == "" {
        fmt.Println("MUDAE_CHANNEL_ID not set in environment")
    } else {
        // attempt to fetch a single message; lack of "Read Message History" or channel read will error
        _, err := sess.ChannelMessages(mudaeChannel, 1, "", "", "")
        if err != nil {
            fmt.Printf("Warning: cannot read message history in channel %s: %v\n", mudaeChannel, err)
        } else {
            fmt.Printf("Read message history OK for channel %s\n", mudaeChannel)
        }
    }

	// Deploy events
	deploy.DeployEvents(sess)

	fmt.Println("Bot is now running")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

