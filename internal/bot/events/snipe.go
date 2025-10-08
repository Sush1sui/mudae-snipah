package events

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// CharacterMeta holds metadata from characters.json
type CharacterMeta struct {
    Rank   int
    Kakera int
}

// charactersMap maps lowercase character name -> metadata
var charactersMap = make(map[string]CharacterMeta)

func init() {
    file, err := os.Open("internal/common/characters.json")
    if err != nil {
        fmt.Println("Error opening characters.json:", err)
        return
    }
    defer file.Close()

    var raw []json.RawMessage
    if err := json.NewDecoder(file).Decode(&raw); err != nil {
        fmt.Println("Error decoding characters.json:", err)
        return
    }

    count := 0
    for _, r := range raw {
        var name string
        var meta CharacterMeta

        // try full object first (rank, name, kakera)
        var full struct {
            Rank   int    `json:"rank"`
            Name   string `json:"name"`
            Kakera int    `json:"kakera"`
        }
        if err := json.Unmarshal(r, &full); err == nil && strings.TrimSpace(full.Name) != "" {
            name = strings.TrimSpace(full.Name)
            meta.Rank = full.Rank
            meta.Kakera = full.Kakera
        } else {
            // try plain string entry
            if err := json.Unmarshal(r, &name); err == nil {
                name = strings.TrimSpace(name)
            } else {
                // try object with "name" field only
                var obj struct {
                    Name string `json:"name"`
                }
                if err := json.Unmarshal(r, &obj); err == nil {
                    name = strings.TrimSpace(obj.Name)
                } else {
                    // unknown entry type, skip
                    continue
                }
            }
        }

        if name == "" {
            continue
        }
        charactersMap[strings.ToLower(name)] = meta
        count++
    }

    fmt.Println("Characters loaded successfully from characters.json with", count, "entries.")
}

func OnSnipeMudae(s *discordgo.Session, m *discordgo.MessageCreate) {
    if m.Author.ID != "432610292342587392" {
        return
    }

    if len(m.Embeds) == 0 || m.Embeds[0] == nil {
        return
    }
    embed := m.Embeds[0]

    if embed == nil || embed.Footer == nil || embed.Author == nil {
        return
    }

    // early-return for footers that are simple counters like "1 / 48"
    footerText := strings.TrimSpace(embed.Footer.Text)
    parts := strings.Split(footerText, "/")
    allInts := len(parts) > 1
    if allInts {
        for _, p := range parts {
            p = strings.TrimSpace(p)
            if p == "" {
                allInts = false
                break
            }
            if _, err := strconv.Atoi(p); err != nil {
                allInts = false
                break
            }
        }
    }
    if allInts {
        // footer looks like an image/page counter (e.g. "1 / 48") — ignore
        return
    }

    if !strings.Contains(strings.ToLower(embed.Footer.Text), "belongs to") {
        // lookup metadata for this character
        charMeta, ok := charactersMap[strings.ToLower(embed.Author.Name)]
        if ok {
            fmt.Println("Top character found:", embed.Author.Name)
            vipUsers := strings.Split(os.Getenv("SNIPER_VIP_USERS"), ",")
            for _, id := range vipUsers {
                id = strings.TrimSpace(id)
                if id == "" {
                    continue
                }
                go func(userID string) {
                    user, err := s.User(userID)
                    if err != nil || user == nil {
                        fmt.Println("Error fetching user:", err)
                        return
                    }
                    dmChannel, err := s.UserChannelCreate(userID)
                    if err != nil {
                        fmt.Println("Error creating DM channel:", err)
						return
                    }

                    // Construct the jump link
                    messageURL := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", m.GuildID, m.ChannelID, m.ID)
                    content := fmt.Sprintf("# A top character `%s` has appeared! Rank: %d, Kakera: %d. Click here to jump to the message: %s",
                        embed.Author.Name, charMeta.Rank, charMeta.Kakera, messageURL)

                    _, err = s.ChannelMessageSendComplex(dmChannel.ID, &discordgo.MessageSend{
                        Content: content,
                        Embed:   embed,
                    })
                    if err != nil {
                        fmt.Println("Error sending DM:", err)
                        return
                    }
                }(id)
            }
            secret := os.Getenv("secret")
            secretInt, err := strconv.Atoi(secret)
            if err != nil {
                fmt.Println("Error converting secret to int:", err)
                return
            }
            time.Sleep(time.Duration(secretInt) * time.Second)
            // get all users who have a role with id=os.Getenv("SNIPER_ROLE_ID") in the guild
            guild, err := s.State.Guild(m.GuildID)
            if err != nil {
                fmt.Println("Error fetching guild from state:", err)
                return
            }
            sniperRoleId := os.Getenv("SNIPER_ROLE_ID")
            if sniperRoleId == "" {
                fmt.Println("SNIPER_ROLE_ID not set in environment variables")
                return
            }
            for _, member := range guild.Members {
                if vipUsers != nil && slices.Contains(vipUsers, member.User.ID) { continue }
                for _, roleID := range member.Roles {
                    if roleID == sniperRoleId {
                        go func(userID string) {
                            user, err := s.User(userID)
                            if err != nil || user == nil {
                                fmt.Println("Error fetching user:", err)
                                return
                            }
                            dmChannel, err := s.UserChannelCreate(userID)
                            if err != nil {
                                fmt.Println("Error creating DM channel:", err)
                                return
                            }

                            // Construct the jump link
                            messageURL := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", m.GuildID, m.ChannelID, m.ID)
                            content := fmt.Sprintf("# A top character `%s` has appeared!. Click here to jump to the message: %s",
                                embed.Author.Name, messageURL)

                            _, err = s.ChannelMessageSendComplex(dmChannel.ID, &discordgo.MessageSend{
                                Content: content,
                                Embed:   embed,
                            })
                            if err != nil {
                                fmt.Println("Error sending DM:", err)
                                return
                            }
                        }(member.User.ID)
                    }
                }
            }
        }
    }
}