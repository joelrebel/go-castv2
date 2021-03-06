package controllers

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/AndreasAbdi/go-castv2/controllers/youtube"
	"github.com/davecgh/go-spew/spew"

	"github.com/AndreasAbdi/go-castv2/api"
	"github.com/AndreasAbdi/go-castv2/primitives"
)

/*
Helps with playing the youtube chromecast app.
See
https://github.com/balloob/pychromecast/blob/master/pychromecast/controllers/youtube.py
and
https://github.com/ur1katz/casttube/blob/master/casttube/YouTubeSession.py.
https://github.com/CBiX/gotubecast/blob/master/main.go
https://github.com/mutantmonkey/youtube-remote/blob/master/remote.py
Essentially, you start a session with the website, and the website/session handles  like any other receiver app.
*/

const loungeIDHeader = "X-YouTube-LoungeId-Token"

var messageTypeGetSessionID = "getMdxSessionStatus"
var responseTypeSessionStatus = "mdxSessionStatus"

//YoutubeController is the controller for the commands unique to the dashcast.
type YoutubeController struct {
	connection *mediaConnection
	screenID   *string
	incoming   chan *string
	session    *youtube.Session
}

//NewYoutubeController is a constructor for a dash cast controller.
func NewYoutubeController(client *primitives.Client, sourceID string, receiver *ReceiverController) *YoutubeController {
	connection := NewMediaConnection(client, receiver, youtubeControllerNamespace, sourceID)
	controller := YoutubeController{
		connection: connection,
		incoming:   make(chan *string, 0),
	}
	connection.OnMessage(responseTypeSessionStatus, controller.onStatus)
	return &controller
}

type youtubeCommand struct {
	primitives.PayloadHeaders
}

//TODO : Do something with the list id
//PlayVideo initializes a new queue and plays the video
func (c *YoutubeController) PlayVideo(videoID string) {
	c.ensureSessionActive()
	c.session.InitializeQueue(videoID, "")
}

//PlayNext adds a video to be played next in the current playlist TODO
func (c *YoutubeController) PlayNext(videoID string) {
	c.ensureSessionActive()
	c.session.PlayNext(videoID)
}

//AddToQueue adds the video to the end of the current playlist TODO
func (c *YoutubeController) AddToQueue(videoID string) {
	c.ensureSessionActive()
	c.session.AddToQueue(videoID)
}

//RemoveFromQueue removes a video from the videoplaylist TODO
func (c *YoutubeController) RemoveFromQueue(videoID string) {
	c.ensureSessionActive()
	c.session.RemoveFromQueue(videoID)
}

func (c *YoutubeController) ensureSessionActive() {
	if c.screenID == nil || c.session == nil {
		err := c.updateScreenID()
		if err != nil {
			return
		}
		c.updateYoutubeSession()
	}
}

func (c *YoutubeController) updateScreenID() error {
	screenID, err := c.getScreenID(time.Second * 5)
	if err != nil {
		spew.Dump("Failed to get screen ID")
		return err
	}
	c.screenID = screenID
	return nil

}

func (c *YoutubeController) updateYoutubeSession() error {
	if c.session == nil {
		c.session = youtube.NewSession(*c.screenID)
	}

	return c.session.StartSession()
}

func (c *YoutubeController) onStatus(message *api.CastMessage) {
	spew.Dump("Got youtube status message")
	response := &youtube.ScreenStatus{}
	err := json.Unmarshal([]byte(*message.PayloadUtf8), response)
	if err != nil {
		spew.Dump("Failed to unmarshal status message:%s - %s", err, *message.PayloadUtf8)
		return
	}
	select {
	case c.incoming <- &response.Data.ScreenID:
		spew.Dump("Delivered status. %v", response)
	case <-time.After(time.Second):
		spew.Dump("Incoming youtube status, but we aren't listening. %v", response)
	}
}

func (c *YoutubeController) getScreenID(timeout time.Duration) (*string, error) {

	waitCh := make(chan bool)
	var screenID *string
	go func() {
		//spew.Dump("Listening for incoming youtube status")
		screenID = <-c.incoming
		waitCh <- true
	}()

	c.connection.Request(
		&primitives.PayloadHeaders{Type: messageTypeGetSessionID},
		0)
	select {
	case <-waitCh:
		return screenID, nil
	case <-time.After(timeout):
		return nil, errors.New("Failed to get screen ID, timed out")
	}
}
