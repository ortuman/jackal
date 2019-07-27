/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pubsubmodel

import (
	"errors"
	"fmt"
	"strconv"

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

const (
	Open             = "open"
	Presence         = "presence"
	Roster           = "roster"
	WhiteList        = "whitelist"
	Publishers       = "publishers"
	Never            = "never"
	OnSub            = "on_sub"
	OnSubAndPresence = "on_sub_and_presence"
)

type Options struct {
	Title                 string
	DeliverNotifications  bool
	DeliverPayloads       bool
	PersistItems          bool
	MaxItems              int64
	ItemExpire            int64
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
	MaxPayloadSize        int64
	Type                  string
	BodyXSLT              string
}

func NewOptionsFromMap(m map[string]string) (*Options, error) {
	opt := &Options{}

	// extract options values
	opt.Title = m[titleOptField]
	opt.DeliverNotifications, _ = strconv.ParseBool(m[deliverNotificationsOptField])
	opt.DeliverPayloads, _ = strconv.ParseBool(m[deliverPayloadsOptField])
	opt.PersistItems, _ = strconv.ParseBool(m[persistItemsOptField])
	opt.MaxItems, _ = strconv.ParseInt(m[maxItemsOptField], 10, 32)
	opt.ItemExpire, _ = strconv.ParseInt(m[itemExpireOptField], 10, 32)
	opt.PurgeOffline, _ = strconv.ParseBool(m[purgeOfflineOptField])
	opt.PresenceBasedDelivery, _ = strconv.ParseBool(m[presenceBasedDeliveryOptField])
	opt.NotificationType = m[notificationTypeOptField]
	opt.NotifyConfig, _ = strconv.ParseBool(m[notifyConfigOptField])
	opt.NotifyDelete, _ = strconv.ParseBool(m[notifyDeleteOptField])
	opt.NotifyRetract, _ = strconv.ParseBool(m[notifyRetractOptField])
	opt.NotifySub, _ = strconv.ParseBool(m[notifySubOptField])
	opt.MaxPayloadSize, _ = strconv.ParseInt(m[maxPayloadSizeOptField], 10, 32)
	opt.Type = m[typeOptField]
	opt.BodyXSLT = m[bodyXSLTOptField]

	// extract options values
	accessModel := m[accessModelOptField]
	switch accessModel {
	case Open, Presence, Roster, WhiteList:
		opt.AccessModel = accessModel
	default:
		return nil, fmt.Errorf("invalid access_model value: %s", accessModel)
	}

	publishModel := m[publishModelOptField]
	switch publishModel {
	case Open, Publishers:
		opt.PublishModel = publishModel
	default:
		return nil, fmt.Errorf("invalid publish_model value: %s", publishModel)
	}

	sendLastPublishedItem := m[sendLastPublishedItemOptField]
	switch sendLastPublishedItem {
	case Never, OnSub, OnSubAndPresence:
		opt.SendLastPublishedItem = sendLastPublishedItem
	default:
		return nil, fmt.Errorf("invalid send_last_published_item value: %s", sendLastPublishedItem)
	}
	return opt, nil
}

func NewOptionsFromForm(form *xep0004.DataForm) (*Options, error) {
	opt := &Options{}
	fields := form.Fields
	if len(fields) == 0 {
		return nil, errors.New("form empty fields")
	}
	// validate form type
	formType := fields.ValueForFieldOfType(xep0004.FormType, xep0004.Hidden)
	if form.Type != xep0004.Submit || formType != nodeConfigNamespace {
		return nil, errors.New("invalid form type")
	}
	// extract options values
	accessModel := fields.ValueForField(accessModelOptField)
	switch accessModel {
	case Open, Presence, Roster, WhiteList:
		opt.AccessModel = accessModel
	default:
		return nil, fmt.Errorf("invalid access_model value: %s", accessModel)
	}

	publishModel := fields.ValueForField(publishModelOptField)
	switch publishModel {
	case Open, Publishers:
		opt.PublishModel = publishModel
	default:
		return nil, fmt.Errorf("invalid publish_model value: %s", publishModel)
	}

	sendLastPublishedItem := fields.ValueForField(sendLastPublishedItemOptField)
	switch sendLastPublishedItem {
	case Never, OnSub, OnSubAndPresence:
		opt.SendLastPublishedItem = sendLastPublishedItem
	default:
		return nil, fmt.Errorf("invalid send_last_published_item value: %s", sendLastPublishedItem)
	}

	opt.Title = fields.ValueForField(titleOptField)
	opt.DeliverNotifications, _ = strconv.ParseBool(fields.ValueForField(deliverNotificationsOptField))
	opt.DeliverPayloads, _ = strconv.ParseBool(fields.ValueForField(deliverPayloadsOptField))
	opt.PersistItems, _ = strconv.ParseBool(fields.ValueForField(persistItemsOptField))
	opt.MaxItems, _ = strconv.ParseInt(fields.ValueForField(maxItemsOptField), 10, 32)
	opt.ItemExpire, _ = strconv.ParseInt(fields.ValueForField(itemExpireOptField), 10, 32)
	opt.PurgeOffline, _ = strconv.ParseBool(fields.ValueForField(purgeOfflineOptField))
	opt.PresenceBasedDelivery, _ = strconv.ParseBool(fields.ValueForField(presenceBasedDeliveryOptField))
	opt.NotificationType = fields.ValueForField(notificationTypeOptField)
	opt.NotifyConfig, _ = strconv.ParseBool(fields.ValueForField(notifyConfigOptField))
	opt.NotifyDelete, _ = strconv.ParseBool(fields.ValueForField(notifyDeleteOptField))
	opt.NotifyRetract, _ = strconv.ParseBool(fields.ValueForField(notifyRetractOptField))
	opt.NotifySub, _ = strconv.ParseBool(fields.ValueForField(notifySubOptField))
	opt.MaxPayloadSize, _ = strconv.ParseInt(fields.ValueForField(maxPayloadSizeOptField), 10, 32)
	opt.Type = fields.ValueForField(typeOptField)
	opt.BodyXSLT = fields.ValueForField(bodyXSLTOptField)

	return opt, nil
}

func (opt *Options) Map() map[string]string {
	m := make(map[string]string)
	m[titleOptField] = opt.Title
	m[deliverNotificationsOptField] = strconv.FormatBool(opt.DeliverNotifications)
	m[deliverPayloadsOptField] = strconv.FormatBool(opt.DeliverPayloads)
	m[persistItemsOptField] = strconv.FormatBool(opt.PersistItems)
	m[maxItemsOptField] = strconv.Itoa(int(opt.MaxItems))
	m[itemExpireOptField] = strconv.Itoa(int(opt.ItemExpire))
	m[accessModelOptField] = string(opt.AccessModel)
	m[publishModelOptField] = string(opt.PublishModel)
	m[purgeOfflineOptField] = strconv.FormatBool(opt.PurgeOffline)
	m[sendLastPublishedItemOptField] = opt.SendLastPublishedItem
	m[presenceBasedDeliveryOptField] = strconv.FormatBool(opt.PresenceBasedDelivery)
	m[notificationTypeOptField] = opt.NotificationType
	m[notifyConfigOptField] = strconv.FormatBool(opt.NotifyConfig)
	m[notifyDeleteOptField] = strconv.FormatBool(opt.NotifyDelete)
	m[notifyRetractOptField] = strconv.FormatBool(opt.NotifyRetract)
	m[notifySubOptField] = strconv.FormatBool(opt.NotifySub)
	m[maxPayloadSizeOptField] = strconv.Itoa(int(opt.MaxPayloadSize))
	m[typeOptField] = opt.Type
	m[bodyXSLTOptField] = opt.BodyXSLT
	return m
}

func (opt *Options) Form() *xep0004.DataForm {
	form := xep0004.DataForm{
		Type: xep0004.Submit,
	}
	// include form type
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    xep0004.FormType,
		Type:   xep0004.Hidden,
		Values: []string{nodeConfigNamespace},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    titleOptField,
		Values: []string{opt.Title},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    deliverNotificationsOptField,
		Values: []string{strconv.FormatBool(opt.DeliverNotifications)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    deliverPayloadsOptField,
		Values: []string{strconv.FormatBool(opt.DeliverPayloads)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    persistItemsOptField,
		Values: []string{strconv.FormatBool(opt.PersistItems)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    maxItemsOptField,
		Values: []string{strconv.Itoa(int(opt.MaxItems))},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    itemExpireOptField,
		Values: []string{strconv.Itoa(int(opt.ItemExpire))},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    accessModelOptField,
		Values: []string{string(opt.AccessModel)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    accessModelOptField,
		Values: []string{string(opt.AccessModel)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    publishModelOptField,
		Values: []string{string(opt.PublishModel)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    purgeOfflineOptField,
		Values: []string{strconv.FormatBool(opt.PurgeOffline)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    sendLastPublishedItemOptField,
		Values: []string{opt.SendLastPublishedItem},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    presenceBasedDeliveryOptField,
		Values: []string{strconv.FormatBool(opt.PresenceBasedDelivery)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    notificationTypeOptField,
		Values: []string{opt.NotificationType},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    notifyConfigOptField,
		Values: []string{strconv.FormatBool(opt.NotifyConfig)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    notifyDeleteOptField,
		Values: []string{strconv.FormatBool(opt.NotifyDelete)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    notifyRetractOptField,
		Values: []string{strconv.FormatBool(opt.NotifyRetract)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    notifySubOptField,
		Values: []string{strconv.FormatBool(opt.NotifySub)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    maxPayloadSizeOptField,
		Values: []string{strconv.Itoa(int(opt.MaxPayloadSize))},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    typeOptField,
		Values: []string{opt.Type},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    bodyXSLTOptField,
		Values: []string{opt.BodyXSLT},
	})
	return &form
}
