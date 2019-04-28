package pubsubmodel

import (
	"errors"

	"github.com/ortuman/jackal/module/xep0004"
)

const nodeConfigNamespace = "http://jabber.org/protocol/pubsub#node_config"

const (
	titleOptField                 = "pubsub#title"
	deliverNotificationsOptField  = "pubsub#deliver_notifications"
	deliverPayloadsOptField       = "pubsub#deliver_payloads"
	persistItemsOptField          = "pubsub#persist_items"
	maxItemsOptField              = "pubsub#max_items"
	itemExpireOptField            = "pubsub#item_expire"
	accessModelOptField           = "pubsub#access_model"
	publishModelOptField          = "pubsub#publish_model"
	purgeOfflineOptField          = "pubsub#purge_offline"
	sendLastPublishedItemOptField = "pubsub#send_last_published_item"
	presenceBasedDeliveryOptField = "pubsub#presence_based_delivery"
	notificationTypeOptField      = "pubsub#notification_type"
	notifyConfigOptField          = "pubsub#notify_config"
	notifyDeleteOptField          = "pubsub#notify_delete"
	notifyRetractOptField         = "pubsub#notify_retract"
	notifySubOptField             = "pubsub#notify_sub"
	maxPayloadSizeOptField        = "pubsub#max_payload_size"
	typeOptField                  = "pubsub#type"
	bodyXSLTOptField              = "pubsub#body_xslt"
)

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

func NewOptions(form *xep0004.DataForm) (*Options, error) {
	opt := &Options{}
	fields := form.Fields
	if len(fields) == 0 {
		return nil, errors.New("form empty fields")
	}
	// validate form type
	formType := fields.ValueForFieldOfType("FORM_TYPE", xep0004.Hidden)
	if form.Type != xep0004.Submit || formType != nodeConfigNamespace {
		return nil, errors.New("invalid form type")
	}
	opt.Title = fields.ValueForField(titleOptField)
	opt.DeliverNotifications = fields.BoolForField(deliverNotificationsOptField)
	opt.DeliverPayloads = fields.BoolForField(deliverPayloadsOptField)
	opt.PersistItems = fields.BoolForField(persistItemsOptField)
	opt.MaxItems = fields.IntForField(maxItemsOptField)
	opt.ItemExpire = fields.IntForField(itemExpireOptField)
	opt.AccessModel = fields.ValueForField(accessModelOptField)
	opt.PublishModel = fields.ValueForField(publishModelOptField)
	opt.PurgeOffline = fields.BoolForField(purgeOfflineOptField)
	opt.SendLastPublishedItem = fields.ValueForField(sendLastPublishedItemOptField)
	opt.PresenceBasedDelivery = fields.BoolForField(presenceBasedDeliveryOptField)
	opt.NotificationType = fields.ValueForField(notificationTypeOptField)
	opt.NotifyConfig = fields.BoolForField(notifyConfigOptField)
	opt.NotifyDelete = fields.BoolForField(notifyDeleteOptField)
	opt.NotifyRetract = fields.BoolForField(notifyRetractOptField)
	opt.NotifySub = fields.BoolForField(notifySubOptField)
	opt.MaxPayloadSize = fields.IntForField(maxPayloadSizeOptField)
	opt.Type = fields.ValueForField(typeOptField)
	opt.BodyXSLT = fields.ValueForField(bodyXSLTOptField)

	// extract form types
	return opt, nil
}
