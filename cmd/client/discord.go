package client

import (
	"github.com/bwmarrin/discordgo"
	"os"
	"syscall"
	"os/signal"
	"YmirBot/proto"
	"google.golang.org/grpc"
	"context"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"strings"
	"YmirBot/pkg/discord"
	"github.com/spf13/viper"
)

type discordBotClient struct {
	Session *discordgo.Session
	Client proto.BotClient
	BotState *BotState
}

func NewDiscordBot() YmirClient {

	log.Println("Creating client...")

	discord, err := discordgo.New("Bot "+viper.GetString("token"))
	if (err != nil) {
		log.Fatalln(err)
		panic(err)
	}

	client := GetClient()
	context := &discordBotContext{}
	context.InitializeEnv()

	return &discordBotClient{discord, client, &BotState{Client: client, Context: context}}
}

func GetClient() proto.BotClient {
	serverAddr := "localhost:9095"
	var opts []grpc.DialOption

	opts = append(opts, grpc.WithInsecure())

	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		panic(err)
	}

	return proto.NewBotClient(conn)
}

func (b *discordBotClient) Run() {

	b.Session.AddHandler(b.BotState.onMessage)
	b.Session.AddHandler(b.BotState.onVoiceStateUpdate)

	err := b.Session.Open()
	if err != nil {
		log.Error("Error opening discord session: %s", err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	b.Session.Close()
}

type discordBotContext struct {

}

func (context *discordBotContext) InitializeEnv() {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("discord")
}

type BotState struct {
	Client proto.BotClient
	Context *discordBotContext
}

func (b *BotState) onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {

	log.Infof("Message received: %s %s %s\n", m.ChannelID, m.Content, m.Author)
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
		return
	}

	if strings.HasPrefix(m.Content, "!") {
		user_meta := discord.GetUser(s, m.Author.ID)

		log.Infof("User meta: %s, %s, %s, %s\n", user_meta.ID, user_meta.Username, user_meta.Email, user_meta.Discriminator)

		resp, err := b.Client.GetResponse(context.TODO(), &proto.BotRequest{Id: uuid.NewV1().String(), Text: m.Content, Name: user_meta.Username})

		if err != nil {
			log.Errorln("Problem getting response from Ymir Server")
		}
		s.ChannelMessageSend(m.ChannelID, resp.Text)
	}
}

func (b *BotState) onVoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {

	log.Infof("VoiceStateUpdate received: %s, %s, %s\n", v.ChannelID, v.UserID, v.SessionID)

	channel_meta := discord.GetChannel(s, v.ChannelID)

	log.Infof("Channel information: %s, %s, %s\n", channel_meta.ID, channel_meta.Name, channel_meta.ParentID)
}

