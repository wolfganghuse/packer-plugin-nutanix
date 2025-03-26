
package nutanix

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/url"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/mitchellh/go-vnc"
	"golang.org/x/net/websocket"
)

type StepVNCConnect struct {
	VNCEnabled         bool
	InsecureConnection bool
}

func (s *StepVNCConnect) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	if !s.VNCEnabled {
		return multistep.ActionContinue
	}
	ui := state.Get("ui").(packersdk.Ui)

	var c *vnc.ClientConn
	var err error

	ui.Say("Connecting to VNC over websocket...")
	c, err = s.ConnectVNCOverWebsocketClient(state)


	if err != nil {
		err = fmt.Errorf("error connecting to VNC: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("vnc_conn", c)
	return multistep.ActionContinue
}

func (s *StepVNCConnect) ConnectVNCOverWebsocketClient(state multistep.StateBag) (*vnc.ClientConn, error) {

	vmUUID := state.Get("vm_uuid").(string)
	websocketUrl := fmt.Sprintf("ws://%s:%d/proxy/%s", "10.122.7.121", 8098, vmUUID)
	log.Printf("[DEBUG] websocket url: %s", websocketUrl)
	u, err := url.Parse(websocketUrl)
	if err != nil {
		err = fmt.Errorf("error parsing websocket url: %s", err)
		state.Put("error", err)
		return nil, err
	}
	origin, err := url.Parse("http://localhost")
	if err != nil {
		err = fmt.Errorf("error parsing websocket origin url: %s", err)
		state.Put("error", err)
		return nil, err
	}

	// Create the websocket connection and set it to a BinaryFrame
	websocketConfig := &websocket.Config{
		Location:  u,
		Origin:    origin,
		TlsConfig: &tls.Config{InsecureSkipVerify: s.InsecureConnection},
		Version:   websocket.ProtocolVersionHybi13,
		Protocol:  []string{"binary"},
	}
	nc, err := websocket.DialConfig(websocketConfig)
	if err != nil {
		err := fmt.Errorf("error dialing: %s", err)
		state.Put("error", err)
		return nil, err
	}
	nc.PayloadType = websocket.BinaryFrame

	// Set up the VNC connection over the websocket.
	ccconfig := &vnc.ClientConfig{
		Auth:      []vnc.ClientAuth{new(vnc.ClientAuthNone)},
		Exclusive: false,
	}
	c, err := vnc.Client(nc, ccconfig)
	if err != nil {
		err := fmt.Errorf("error setting the VNC over websocket client: %s", err)
		state.Put("error", err)
		return nil, err
	}
	return c, nil
}


func (s *StepVNCConnect) Cleanup(multistep.StateBag) {}