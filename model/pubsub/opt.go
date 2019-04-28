package pubsubmodel

import (
	"errors"

	"github.com/ortuman/jackal/module/xep0004"
)

const nodeConfigNamespace = "http://jabber.org/protocol/pubsub#node_config"

type Options struct {
	Title                 string
	DeliverNotifications  bool
	DeliverPayloads       bool
	PersistItems          bool
	MaxItems              int
	ItemExpire            int
	AccessModel           string
	PublishModel          string
	PurgeOffline          bool
	SendLastPublishedItem string
	PresenceBasedDelivery bool
	NotificationType      string
	NotifyConfig          bool
	NotifyDelete          bool
	NotifyRetract         bool
	NotifySub             bool
	MaxPayloadSize        int
	Type                  string
	BodyXSLT              string
}

func NewOptions(form xep0004.DataForm) (*Options, error) {
	opt := &Options{}
	fields := form.Fields
	if len(fields) == 0 {
		return nil, errors.New("form empty fields")
	}
	// validate form type
	if formType := fields.ValueForFieldOfType("FORM_TYPE", xep0004.Hidden); formType != nodeConfigNamespace {
		return nil, errors.New("invalid form type")
	}

	return opt, nil
}
