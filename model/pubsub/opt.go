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
	titleFieldVar                 = "pubsub#title"
	deliverNotificationsFieldVar  = "pubsub#deliver_notifications"
	deliverPayloadsFieldVar       = "pubsub#deliver_payloads"
	persistItemsFieldVar          = "pubsub#persist_items"
	maxItemsFieldVar              = "pubsub#max_items"
	accessModelFieldVar           = "pubsub#access_model"
	publishModelFieldVar          = "pubsub#publish_model"
	sendLastPublishedItemFieldVar = "pubsub#send_last_published_item"
	notificationTypeFieldVar      = "pubsub#notification_type"
	notifyConfigFieldVar          = "pubsub#notify_config"
	notifyDeleteFieldVar          = "pubsub#notify_delete"
	notifyRetractFieldVar         = "pubsub#notify_retract"
	notifySubFieldVar             = "pubsub#notify_sub"
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
	AccessModel           string
	PublishModel          string
	SendLastPublishedItem string
	NotificationType      string
	NotifyConfig          bool
	NotifyDelete          bool
	NotifyRetract         bool
	NotifySub             bool
}

func NewOptionsFromMap(m map[string]string) (*Options, error) {
	opt := &Options{}

	// extract options values
	opt.Title = m[titleFieldVar]
	opt.DeliverNotifications, _ = strconv.ParseBool(m[deliverNotificationsFieldVar])
	opt.DeliverPayloads, _ = strconv.ParseBool(m[deliverPayloadsFieldVar])
	opt.PersistItems, _ = strconv.ParseBool(m[persistItemsFieldVar])
	opt.MaxItems, _ = strconv.ParseInt(m[maxItemsFieldVar], 10, 32)
	opt.NotificationType = m[notificationTypeFieldVar]
	opt.NotifyConfig, _ = strconv.ParseBool(m[notifyConfigFieldVar])
	opt.NotifyDelete, _ = strconv.ParseBool(m[notifyDeleteFieldVar])
	opt.NotifyRetract, _ = strconv.ParseBool(m[notifyRetractFieldVar])
	opt.NotifySub, _ = strconv.ParseBool(m[notifySubFieldVar])

	// extract options values
	accessModel := m[accessModelFieldVar]
	switch accessModel {
	case Open, Presence, Roster, WhiteList:
		opt.AccessModel = accessModel
	default:
		return nil, fmt.Errorf("invalid access_model value: %s", accessModel)
	}

	publishModel := m[publishModelFieldVar]
	switch publishModel {
	case Open, Publishers:
		opt.PublishModel = publishModel
	default:
		return nil, fmt.Errorf("invalid publish_model value: %s", publishModel)
	}

	sendLastPublishedItem := m[sendLastPublishedItemFieldVar]
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
	accessModel := fields.ValueForField(accessModelFieldVar)
	switch accessModel {
	case Open, Presence, Roster, WhiteList:
		opt.AccessModel = accessModel
	default:
		return nil, fmt.Errorf("invalid access_model value: %s", accessModel)
	}

	publishModel := fields.ValueForField(publishModelFieldVar)
	switch publishModel {
	case Open, Publishers:
		opt.PublishModel = publishModel
	default:
		return nil, fmt.Errorf("invalid publish_model value: %s", publishModel)
	}

	sendLastPublishedItem := fields.ValueForField(sendLastPublishedItemFieldVar)
	switch sendLastPublishedItem {
	case Never, OnSub, OnSubAndPresence:
		opt.SendLastPublishedItem = sendLastPublishedItem
	default:
		return nil, fmt.Errorf("invalid send_last_published_item value: %s", sendLastPublishedItem)
	}

	opt.Title = fields.ValueForField(titleFieldVar)
	opt.DeliverNotifications, _ = strconv.ParseBool(fields.ValueForField(deliverNotificationsFieldVar))
	opt.DeliverPayloads, _ = strconv.ParseBool(fields.ValueForField(deliverPayloadsFieldVar))
	opt.PersistItems, _ = strconv.ParseBool(fields.ValueForField(persistItemsFieldVar))
	opt.MaxItems, _ = strconv.ParseInt(fields.ValueForField(maxItemsFieldVar), 10, 32)
	opt.NotificationType = fields.ValueForField(notificationTypeFieldVar)
	opt.NotifyConfig, _ = strconv.ParseBool(fields.ValueForField(notifyConfigFieldVar))
	opt.NotifyDelete, _ = strconv.ParseBool(fields.ValueForField(notifyDeleteFieldVar))
	opt.NotifyRetract, _ = strconv.ParseBool(fields.ValueForField(notifyRetractFieldVar))
	opt.NotifySub, _ = strconv.ParseBool(fields.ValueForField(notifySubFieldVar))

	return opt, nil
}

func (opt *Options) Map() map[string]string {
	m := make(map[string]string)
	m[titleFieldVar] = opt.Title
	m[deliverNotificationsFieldVar] = strconv.FormatBool(opt.DeliverNotifications)
	m[deliverPayloadsFieldVar] = strconv.FormatBool(opt.DeliverPayloads)
	m[persistItemsFieldVar] = strconv.FormatBool(opt.PersistItems)
	m[maxItemsFieldVar] = strconv.Itoa(int(opt.MaxItems))
	m[accessModelFieldVar] = string(opt.AccessModel)
	m[publishModelFieldVar] = string(opt.PublishModel)
	m[sendLastPublishedItemFieldVar] = opt.SendLastPublishedItem
	m[notificationTypeFieldVar] = opt.NotificationType
	m[notifyConfigFieldVar] = strconv.FormatBool(opt.NotifyConfig)
	m[notifyDeleteFieldVar] = strconv.FormatBool(opt.NotifyDelete)
	m[notifyRetractFieldVar] = strconv.FormatBool(opt.NotifyRetract)
	m[notifySubFieldVar] = strconv.FormatBool(opt.NotifySub)
	return m
}

func (opt *Options) Form() *xep0004.DataForm {
	form := xep0004.DataForm{
		Type: xep0004.Form,
	}
	// include form type
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    xep0004.FormType,
		Type:   xep0004.Hidden,
		Values: []string{nodeConfigNamespace},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    titleFieldVar,
		Type:   xep0004.TextSingle,
		Label:  "Node title",
		Values: []string{opt.Title},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    deliverNotificationsFieldVar,
		Type:   xep0004.Boolean,
		Label:  "Whether to deliver event notifications",
		Values: []string{strconv.FormatBool(opt.DeliverNotifications)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    deliverPayloadsFieldVar,
		Type:   xep0004.Boolean,
		Label:  "Whether to deliver payloads with event notifications",
		Values: []string{strconv.FormatBool(opt.DeliverPayloads)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    persistItemsFieldVar,
		Type:   xep0004.Boolean,
		Label:  "Whether to persist items to storage",
		Values: []string{strconv.FormatBool(opt.PersistItems)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    maxItemsFieldVar,
		Type:   xep0004.Boolean,
		Label:  "Max number of items to persist",
		Values: []string{strconv.FormatInt(opt.MaxItems, 10)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    accessModelFieldVar,
		Type:   xep0004.ListSingle,
		Values: []string{opt.AccessModel},
		Label:  "Specify the subscriber model",
		Options: []xep0004.Option{
			{Label: "Open", Value: Open},
			{Label: "Presence Sharing", Value: Presence},
			{Label: "Roster Groups", Value: Roster},
			{Label: "Whitelist", Value: WhiteList},
		},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    publishModelFieldVar,
		Type:   xep0004.ListSingle,
		Label:  "Specify the publisher model",
		Values: []string{opt.PublishModel},
		Options: []xep0004.Option{
			{Label: "Only publishers may publish", Value: Publishers},
			{Label: "Anyone may publish", Value: Open},
		},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    sendLastPublishedItemFieldVar,
		Type:   xep0004.ListSingle,
		Label:  "When to send the last published item",
		Values: []string{opt.SendLastPublishedItem},
		Options: []xep0004.Option{
			{Label: "Never", Value: Never},
			{Label: "When a new subscription is processed", Value: OnSub},
			{Label: "When a new subscription is processed and whenever a subscriber comes online", Value: OnSubAndPresence},
		},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    notifyConfigFieldVar,
		Type:   xep0004.Boolean,
		Label:  "Notify subscribers when the node configuration changes",
		Values: []string{strconv.FormatBool(opt.NotifyConfig)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    notifyDeleteFieldVar,
		Type:   xep0004.Boolean,
		Label:  "Notify subscribers when the node is deleted",
		Values: []string{strconv.FormatBool(opt.NotifyDelete)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    notifyRetractFieldVar,
		Type:   xep0004.Boolean,
		Label:  "Notify subscribers when items are removed from the node",
		Values: []string{strconv.FormatBool(opt.NotifyRetract)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    notifySubFieldVar,
		Type:   xep0004.Boolean,
		Label:  "Notify owners about new subscribers and unsubscribes",
		Values: []string{strconv.FormatBool(opt.NotifySub)},
	})
	return &form
}

func (opt *Options) SubmitForm() *xep0004.DataForm {
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
		Var:    titleFieldVar,
		Values: []string{opt.Title},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    deliverNotificationsFieldVar,
		Values: []string{strconv.FormatBool(opt.DeliverNotifications)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    deliverPayloadsFieldVar,
		Values: []string{strconv.FormatBool(opt.DeliverPayloads)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    persistItemsFieldVar,
		Values: []string{strconv.FormatBool(opt.PersistItems)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    maxItemsFieldVar,
		Values: []string{strconv.Itoa(int(opt.MaxItems))},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    accessModelFieldVar,
		Values: []string{opt.AccessModel},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    accessModelFieldVar,
		Values: []string{opt.AccessModel},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    publishModelFieldVar,
		Values: []string{opt.PublishModel},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    sendLastPublishedItemFieldVar,
		Values: []string{opt.SendLastPublishedItem},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    notificationTypeFieldVar,
		Values: []string{opt.NotificationType},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    notifyConfigFieldVar,
		Values: []string{strconv.FormatBool(opt.NotifyConfig)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    notifyDeleteFieldVar,
		Values: []string{strconv.FormatBool(opt.NotifyDelete)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    notifyRetractFieldVar,
		Values: []string{strconv.FormatBool(opt.NotifyRetract)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    notifySubFieldVar,
		Values: []string{strconv.FormatBool(opt.NotifySub)},
	})
	return &form
}
