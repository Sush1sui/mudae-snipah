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

// add fast-path globals for env + DM queue
var (
    vipUsers    []string
    vipSet      map[string]struct{}
    sniperRole  string
    secretDelay time.Duration

    dmQueue    chan dmJob
    dmWorkers  = 6
    queueSize  = 256
)

type dmJob struct {
    session *discordgo.Session
    userID  string
    content string
    embed   *discordgo.MessageEmbed
}

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

    // parse environment once
    rawVIP := os.Getenv("SNIPER_VIP_USERS")
    vipSet = make(map[string]struct{})
    if rawVIP != "" {
        for _, id := range strings.Split(rawVIP, ",") {
            id = strings.TrimSpace(id)
            if id == "" {
                continue
            }
            vipUsers = append(vipUsers, id)
            vipSet[id] = struct{}{}
        }
    }
    sniperRole = strings.TrimSpace(os.Getenv("SNIPER_ROLE_ID"))
    sec := os.Getenv("secret")
    if sec == "" {
        sec = "5"
    }
    if n, err := strconv.Atoi(sec); err == nil {
        secretDelay = time.Duration(n) * time.Second
    } else {
        secretDelay = 5 * time.Second
    }

    // start DM worker pool
    dmQueue = make(chan dmJob, queueSize)
    for i := 0; i < dmWorkers; i++ {
        go func(id int) {
            for job := range dmQueue {
                dmCh, err := job.session.UserChannelCreate(job.userID)
                if err != nil {
                    fmt.Println("DM worker: create channel error:", err)
                    continue
                }
                if job.content != "" {
                    if _, err := job.session.ChannelMessageSend(dmCh.ID, job.content); err != nil {
                        fmt.Println("DM worker: send text error:", err)
                        continue
                    }
                }
                if job.embed != nil {
                    if _, err := job.session.ChannelMessageSendComplex(dmCh.ID, &discordgo.MessageSend{Embed: job.embed}); err != nil {
                        fmt.Println("DM worker: send embed error:", err)
                        continue
                    }
                }
            }
        }(i)
    }
}

func OnSnipeMudae(s *discordgo.Session, m *discordgo.MessageCreate) {
    // quick early checks
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

    // skip "1 / 48" style footers
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
    if allInts { return }

    fmt.Printf("Character: %s", embed.Author.Name)

    // require "belongs to" (adjust if your logic differs)
    if !strings.Contains(strings.ToLower(footerText), "belongs to") { return }

    // lookup metadata (use preloaded map)
    key := strings.ToLower(strings.TrimSpace(embed.Author.Name))
    charMeta, ok := charactersMap[key]
    if !ok { return }

    // build notification content once
    messageURL := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", m.GuildID, m.ChannelID, m.ID)
    content := fmt.Sprintf("A top character `%s` has appeared! Rank: %d, Kakera: %d. Jump: %s",
        embed.Author.Name, charMeta.Rank, charMeta.Kakera, messageURL)

    // enqueue VIP DMs very quickly
    for _, id := range vipUsers {
        enqueueDM(s, id, content, embed)
    }

    // schedule role notifications after delay without blocking handler
    time.AfterFunc(secretDelay, func() {
        guild, err := s.State.Guild(m.GuildID)
        if err != nil {
            fmt.Println("Error fetching guild from state:", err)
            return
        }
        for _, member := range guild.Members {
            // skip VIPs via map O(1)
            if _, ok := vipSet[member.User.ID]; ok {
                continue
            }
            // check role membership
            if slices.Contains(member.Roles, sniperRole) {
                roleContent := fmt.Sprintf("Top character `%s` appeared — %s", embed.Author.Name, messageURL)
                enqueueDM(s, member.User.ID, roleContent, embed)
            }
        }
    })
}


// helper: enqueue DM non-blocking
func enqueueDM(s *discordgo.Session, userID, content string, embed *discordgo.MessageEmbed) {
    select {
    case dmQueue <- dmJob{session: s, userID: userID, content: content, embed: embed}:
    default:
        // queue full — drop and log to avoid blocking event loop
        fmt.Println("DM queue full; dropped DM for", userID)
    }
}