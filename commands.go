package main

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// ID for the CTF text channel category
const ctfCategoryID = "801259574317416479"

// Define our ApplicationCommands
var (
	// Create an array of ApplicationCommand structs to register the definitions of our commands
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "ping",
			Description: "ping-command",
		},
		{
			Name:        "ctf",
			Type:        discordgo.ChatApplicationCommand,
			Description: "Parent command for the CTF group",
			// Subcommands for 'ctf'
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "create",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "Create a CTF",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "ctf-name",
							Description: "CTF name",
							Required:    true,
						},
					},
				},
				{
					Name:        "join",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "Join a CTF",
				},
			},
		},
	}

	// Create a map of <CommandName>:<HandlerFunction> for each command. Each command will
	// correspond to a first-class function that will handle the command's usage upon invocation
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		// Ping command
		"ping": pingCommandCallback,
		// 'ctf' command group. The function is defined below for cleanliness
		"ctf": ctfCommandCallback,
	}

	// Define handlers for message components. That is to say, what will be executed when a
	// component is interacted with.
	componentsHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"ctf_join": joinCTFButtonCallback,
	}
)

// Ping command handler
func pingCommandCallback(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Pong!",
		},
	})
}

// This function handles the response action(s) for the 'ctf' group of ApplicationCommands
func ctfCommandCallback(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var respContent string
	var ctfName string
	var respComponents []discordgo.MessageComponent
	data := i.ApplicationCommandData()

	// Check which subcommand was called
	switch data.Options[0].Name {
	case "create":
		ctfName = data.Options[0].Options[0].StringValue()
		log.Printf("New CTF Name given: %s\n", ctfName)

		// Create the new role for the CTF
		newRole, err := s.GuildRoleCreate(GlobalConfig.GuildID)
		if err != nil {
			respContent = "Could not create new guild role: " + err.Error()
		} else {
			s.GuildRoleEdit(GlobalConfig.GuildID, newRole.ID, ctfName, 0, false, 0, true)
			respContent = ctfName
		}

		// Reply with a button to allow quickly joining the CTF
		actionRow := discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Join " + ctfName,
					Style:    discordgo.SuccessButton,
					Disabled: false,
					CustomID: "ctf_join",
				},
			},
		}
		respComponents = append(respComponents, actionRow)

		// Create a channel for the CTF that is locked to the created role
		targetGuild, _ := s.Guild(i.GuildID)
		everyoneID := targetGuild.Roles[0].ID // Per my assumption, @everyone is the first role - Jack

		s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
			Name:     ctfName,
			Type:     discordgo.ChannelTypeGuildText,
			Topic:    "Channel for " + ctfName,
			ParentID: ctfCategoryID,
			PermissionOverwrites: []*discordgo.PermissionOverwrite{
				{ // deny everyone read perms
					ID:    everyoneID,
					Type:  0,
					Deny:  discordgo.PermissionViewChannel,
					Allow: 0,
				},
				{ // allow the new role to view and send messages
					ID:    newRole.ID,
					Type:  0,
					Deny:  0,
					Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages,
				},
			},
		})
	case "join":
		guild, err := s.Guild(i.GuildID)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   1 << 6,
					Content: "Couldn't find the calling Guild",
				},
			})
			return
		}
		ctfName := i.Message.Content
		callingUser := i.Member.User
		log.Printf("Adding user %s to CTF %s\n", callingUser.Username, ctfName)

		// Find the role of the CTF and add the user to it
		var ctfRole *discordgo.Role
		for _, r := range guild.Roles {
			if r.Name == ctfName {
				ctfRole = r
			}
		}
		// Check if the role was ever found and correct if not
		if ctfRole == nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   1 << 6,
					Content: "Role " + ctfName + " does not exist. Try creating it, first!",
				},
			})
			return
		}

		s.GuildMemberRoleAdd(guild.ID, callingUser.ID, ctfRole.ID)

		// Reply with success
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   1 << 6,
				Content: "Added user " + callingUser.Username + " to role " + ctfRole.Name,
			},
		})
	}

	// Send back the status reply
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    respContent,
			Components: respComponents,
		},
	})
	if err != nil {
		log.Print(err)
	}
}

func joinCTFButtonCallback(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guild, err := s.Guild(i.GuildID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   1 << 6,
				Content: "Couldn't find the calling Guild",
			},
		})
		return
	}
	ctfName := i.Message.Content
	callingUser := i.Member.User
	log.Printf("Adding user %s to CTF %s\n", callingUser.Username, ctfName)

	// Find the role of the CTF and add the user to it
	var ctfRole *discordgo.Role
	for _, r := range guild.Roles {
		if r.Name == ctfName {
			ctfRole = r
		}
	}
	// Check if the role was ever found and correct if not
	if ctfRole == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   1 << 6,
				Content: "Role " + ctfName + " does not exist. Try creating it, first!",
			},
		})
		return
	}

	s.GuildMemberRoleAdd(guild.ID, callingUser.ID, ctfRole.ID)

	// Reply with success
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   1 << 6,
			Content: "Added user " + callingUser.Username + " to role " + ctfRole.Name,
		},
	})
}
