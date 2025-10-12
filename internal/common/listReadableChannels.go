package common

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// ListReadableChannels prints all guild text channels the bot can view and read history in.
func ListReadableChannels(s *discordgo.Session) {
    if s.State == nil || s.State.User == nil {
        fmt.Println("Session state or user not ready")
        return
    }
    botID := s.State.User.ID

    for _, g := range s.State.Guilds {
        var channels []*discordgo.Channel
        // prefer state channels if present
        if g != nil && len(g.Channels) > 0 {
            channels = g.Channels
        } else {
            // fallback to API fetch
            chs, err := s.GuildChannels(g.ID)
            if err != nil {
                fmt.Printf("Failed to fetch channels for guild %s (%s): %v\n", g.Name, g.ID, err)
                continue
            }
            channels = chs
        }

        for _, ch := range channels {
            // only consider guild text-like channels
            if ch == nil {
                continue
            }
            if ch.Type != discordgo.ChannelTypeGuildText && ch.Type != discordgo.ChannelTypeGuildNews {
                continue
            }

            perms := computeChannelPermissions(s, g, ch, botID)
            canView := perms&discordgo.PermissionViewChannel != 0
            canReadHistory := perms&discordgo.PermissionReadMessageHistory != 0

            if canView && canReadHistory {
                fmt.Printf("Readable: guild=%s(%s) channel=%s(%s)\n", g.Name, g.ID, ch.Name, ch.ID)
            }
        }
    }
}

func computeChannelPermissions(s *discordgo.Session, g *discordgo.Guild, ch *discordgo.Channel, userID string) int {
    // get member (prefer cache)
    var member *discordgo.Member
    if s.State != nil {
        member, _ = s.State.Member(g.ID, userID)
    }
    if member == nil {
        m, err := s.GuildMember(g.ID, userID)
        if err != nil {
            return 0
        }
        member = m
    }

    // owner has implicit full access to the guild
    if userID == g.OwnerID {
        // ensure the bits we care about are set (Administrator bypasses denies)
        return discordgo.PermissionAdministrator | discordgo.PermissionViewChannel | discordgo.PermissionReadMessageHistory
    }

    // build role map for quick lookup
    roleMap := make(map[string]*discordgo.Role, len(g.Roles))
    for _, r := range g.Roles {
        roleMap[r.ID] = r
    }

    // start with @everyone role perms (role ID == guild ID)
    var perms int
    for _, r := range g.Roles {
        if r.ID == g.ID {
            perms = int(r.Permissions)
            break
        }
    }

    // add member roles
    for _, rid := range member.Roles {
        if role, ok := roleMap[rid]; ok {
            perms |= int(role.Permissions)
        }
    }

    // admin shortcut: if member has Administrator, they bypass channel denies
    if perms&discordgo.PermissionAdministrator != 0 {
        return perms
    }

    // apply channel overwrites:
    // 1) @everyone (overwrite.ID == guild.ID)
    // 2) role overwrites (accumulate deny/allow across member's roles)
    // 3) member-specific overwrite
    // Overwrite effect: perms = (perms &^ deny) | allow

    // helper to apply a single overwrite
    apply := func(cur int, allow int, deny int) int {
        cur &^= deny
        cur |= allow
        return cur
    }

    // 1) @everyone overwrite
    var roleDeny, roleAllow int
    for _, ov := range ch.PermissionOverwrites {
        if ov.ID == g.ID { // everyone
            perms = apply(perms, int(ov.Allow), int(ov.Deny))
            break
        }
    }

    // 2) role overwrites (aggregate)
    roleDeny = 0
    roleAllow = 0
    for _, ov := range ch.PermissionOverwrites {
        // skip non-role overwrites (some libs use Type field; checking membership by roleMap is robust)
        if _, ok := roleMap[ov.ID]; !ok {
            continue
        }
        // if member has this role, accumulate
        for _, rid := range member.Roles {
            if ov.ID == rid {
                roleDeny |= int(ov.Deny)
                roleAllow |= int(ov.Allow)
                break
            }
        }
    }
    if roleDeny != 0 || roleAllow != 0 {
        perms = apply(perms, roleAllow, roleDeny)
    }

    // 3) member overwrite
    for _, ov := range ch.PermissionOverwrites {
        if ov.ID == userID {
            perms = apply(perms, int(ov.Allow), int(ov.Deny))
            break
        }
    }

    return perms
}