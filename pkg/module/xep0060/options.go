// Copyright 2022 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xep0060

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/jackal-xmpp/stravaganza"
	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"
	"github.com/ortuman/jackal/pkg/module/xep0004"
	"google.golang.org/protobuf/proto"
)

const (
	languageFormKey              = "language"
	titleFormKey                 = "title"
	descriptionFormKey           = "description"
	deliverPayloadsFormKey       = "deliver_payloads"
	deliverNotificationsFormKey  = "deliver_notifications"
	notifyConfigFormKey          = "notify_config"
	notifyDeleteFormKey          = "notify_delete"
	notifyRetractFormKey         = "notify_retract"
	notifySubFormKey             = "notify_sub"
	persistItemsFormKey          = "persist_items"
	maxItemsFormKey              = "max_items"
	itemExpireFormKey            = "item_expire"
	subscribeFormKey             = "subscribe"
	rosterGroupsAllowedFormKey   = "roster_groups_allowed"
	publishModelFormKey          = "publish_model"
	purgeOffLineFormKey          = "purge_offline"
	maxPayloadSizeFormKey        = "max_payload_size"
	presenceBasedDeliveryFormKey = "presence_based_delivery"
	accessModelFormKey           = "access_model"
	sendLastPublishedItemFormKey = "send_last_published_item"
	notificationTypeFormKey      = "notification_type"
	typeFormKey                  = "type"
	bodyXSLTFormKey              = "body_xslt"
	dataFormXSLTFormKey          = "dataform_xslt"
)

func formToOptions(from *pubsubmodel.Options, x stravaganza.Element) (*pubsubmodel.Options, error) {
	fm, err := xep0004.NewFormFromElement(x)
	if err != nil {
		return nil, err
	}

	fmType := fm.Fields.ValueForFieldOfType(xep0004.FormType, xep0004.Hidden)
	if fm.Type != xep0004.Submit || fmType != pubSubNodeConfigNS() {
		return nil, errors.New("unexpected node config form type value")
	}
	fields := fm.Fields

	opts := proto.Clone(from).(*pubsubmodel.Options)

	if val := fields.ValueForField(pubSubFormKey(accessModelFormKey)); len(val) > 0 {
		switch val {
		case "open", "whitelist", "presence", "roster", "authorize":
			opts.AccessModel = val

		default:
			return nil, fmt.Errorf("unrecognized access model value: %s", val)
		}
	}

	if val := fields.ValueForField(pubSubFormKey(sendLastPublishedItemFormKey)); len(val) > 0 {
		switch val {
		case "never", "on_sub_and_presence", "on_sub_only":
			opts.SendLastPublishedItem = val

		default:
			return nil, fmt.Errorf("unrecognized send last published item value: %s", val)
		}
	}

	if val := fields.ValueForField(pubSubFormKey(languageFormKey)); len(val) > 0 {
		opts.Language = val
	}
	if val := fields.ValueForField(pubSubFormKey(titleFormKey)); len(val) > 0 {
		opts.Title = val
	}
	if val := fields.ValueForField(pubSubFormKey(descriptionFormKey)); len(val) > 0 {
		opts.Description = val
	}
	if val, _ := strconv.ParseBool(fields.ValueForField(pubSubFormKey(deliverPayloadsFormKey))); val {
		opts.DeliverPayloads = val
	}
	if val, _ := strconv.ParseBool(fields.ValueForField(pubSubFormKey(deliverNotificationsFormKey))); val {
		opts.DeliverNotifications = val
	}
	if val, _ := strconv.ParseBool(fields.ValueForField(pubSubFormKey(notifyConfigFormKey))); val {
		opts.NotifyConfig = val
	}
	if val, _ := strconv.ParseBool(fields.ValueForField(pubSubFormKey(notifyDeleteFormKey))); val {
		opts.NotifyDelete = val
	}
	if val, _ := strconv.ParseBool(fields.ValueForField(pubSubFormKey(notifyRetractFormKey))); val {
		opts.NotifyRetract = val
	}
	if val, _ := strconv.ParseBool(fields.ValueForField(pubSubFormKey(notifySubFormKey))); val {
		opts.NotifySub = val
	}
	if val, _ := strconv.ParseBool(fields.ValueForField(pubSubFormKey(persistItemsFormKey))); val {
		opts.PersistItems = val
	}
	if val, _ := strconv.ParseInt(fields.ValueForField(pubSubFormKey(maxItemsFormKey)), 10, 64); val > 0 {
		opts.MaxItems = val
	}
	if val, _ := strconv.ParseUint(fields.ValueForField(pubSubFormKey(itemExpireFormKey)), 10, 64); val > 0 {
		opts.ItemExpire = val
	}
	if val, _ := strconv.ParseBool(fields.ValueForField(pubSubFormKey(subscribeFormKey))); val {
		opts.Subscribe = val
	}
	if val := fields.ValuesForField(pubSubFormKey(rosterGroupsAllowedFormKey)); len(val) > 0 {
		opts.RosterGroupsAllowed = val
	}
	if val := fields.ValueForField(pubSubFormKey(publishModelFormKey)); len(val) > 0 {
		opts.PublishModel = val
	}
	if val, _ := strconv.ParseBool(fields.ValueForField(pubSubFormKey(purgeOffLineFormKey))); val {
		opts.PurgeOffline = val
	}
	if val, _ := strconv.ParseInt(fields.ValueForField(pubSubFormKey(maxPayloadSizeFormKey)), 10, 64); val > 0 {
		opts.MaxPayloadSize = val
	}
	if val, _ := strconv.ParseBool(fields.ValueForField(pubSubFormKey(presenceBasedDeliveryFormKey))); val {
		opts.PresenceBasedDelivery = val
	}
	if val := fields.ValueForField(pubSubFormKey(notificationTypeFormKey)); len(val) > 0 {
		switch val {
		case stravaganza.NormalType, stravaganza.HeadlineType:
			opts.NotificationType = val

		default:
			return nil, fmt.Errorf("unrecognized notification type value: %s", val)
		}
	}
	if val := fields.ValueForField(pubSubFormKey(typeFormKey)); len(val) > 0 {
		opts.Type = val
	}
	if val := fields.ValueForField(pubSubFormKey(bodyXSLTFormKey)); len(val) > 0 {
		opts.BodyXslt = val
	}
	if val := fields.ValueForField(pubSubFormKey(dataFormXSLTFormKey)); len(val) > 0 {
		opts.DataformXslt = val
	}
	return opts, nil
}

func optionsToForm(opts *pubsubmodel.Options, formType string) *xep0004.DataForm {
	f := xep0004.DataForm{
		Type: formType,
	}

	f.Fields = append(f.Fields, xep0004.Field{
		Var:  xep0004.FormType,
		Type: xep0004.Hidden,
		Values: []string{
			pubSubNodeConfigNS(),
		},
	})

	// language
	field := xep0004.Field{
		Var:   pubSubFormKey(languageFormKey),
		Type:  xep0004.TextSingle,
		Label: "The default language of the node",
	}
	if len(opts.Language) > 0 {
		field.Values = []string{opts.Language}
	}
	f.Fields = append(f.Fields, field)

	// title
	field = xep0004.Field{
		Var:   pubSubFormKey(titleFormKey),
		Type:  xep0004.TextSingle,
		Label: "A friendly name for the node",
	}
	if len(opts.Title) > 0 {
		field.Values = []string{opts.Title}
	}
	f.Fields = append(f.Fields, field)

	// description
	field = xep0004.Field{
		Var:   pubSubFormKey(descriptionFormKey),
		Type:  xep0004.TextSingle,
		Label: "A description of the node",
	}
	if len(opts.Description) > 0 {
		field.Values = []string{opts.Description}
	}
	f.Fields = append(f.Fields, field)

	// deliver_payloads
	field = xep0004.Field{
		Var:   pubSubFormKey(deliverPayloadsFormKey),
		Type:  xep0004.Boolean,
		Label: "Whether to deliver payloads with event notifications",
		Values: []string{
			strconv.FormatBool(opts.DeliverPayloads),
		},
	}
	f.Fields = append(f.Fields, field)

	// deliver_notifications
	field = xep0004.Field{
		Var:   pubSubFormKey(deliverNotificationsFormKey),
		Type:  xep0004.Boolean,
		Label: "Whether to deliver event notifications",
		Values: []string{
			strconv.FormatBool(opts.DeliverNotifications),
		},
	}
	f.Fields = append(f.Fields, field)

	// notify_config
	field = xep0004.Field{
		Var:   pubSubFormKey(notifyConfigFormKey),
		Type:  xep0004.Boolean,
		Label: "Whether to notify subscribers when the node configuration changes",
		Values: []string{
			strconv.FormatBool(opts.NotifyConfig),
		},
	}
	f.Fields = append(f.Fields, field)

	// notify_delete
	field = xep0004.Field{
		Var:   pubSubFormKey(notifyDeleteFormKey),
		Type:  xep0004.Boolean,
		Label: "Whether to notify subscribers when the node is deleted",
		Values: []string{
			strconv.FormatBool(opts.NotifyDelete),
		},
	}
	f.Fields = append(f.Fields, field)

	// notify_retract
	field = xep0004.Field{
		Var:   pubSubFormKey(notifyRetractFormKey),
		Type:  xep0004.Boolean,
		Label: "Whether to notify subscribers when items are removed from the node",
		Values: []string{
			strconv.FormatBool(opts.NotifyRetract),
		},
	}
	f.Fields = append(f.Fields, field)

	// notify_sub
	field = xep0004.Field{
		Var:   pubSubFormKey(notifySubFormKey),
		Type:  xep0004.Boolean,
		Label: "Whether to notify owners about new subscribers and unsubscribes",
		Values: []string{
			strconv.FormatBool(opts.NotifySub),
		},
	}
	f.Fields = append(f.Fields, field)

	// persist_items
	field = xep0004.Field{
		Var:   pubSubFormKey(persistItemsFormKey),
		Type:  xep0004.Boolean,
		Label: "Whether to persist items to storage",
		Values: []string{
			strconv.FormatBool(opts.PersistItems),
		},
	}
	f.Fields = append(f.Fields, field)

	// max_items
	field = xep0004.Field{
		Var:   pubSubFormKey(maxItemsFormKey),
		Type:  xep0004.TextSingle,
		Label: "The maximum number of items to persist. `max` for no specific limit other than a server imposed maximum.",
	}
	if opts.MaxItems > 0 {
		field.Values = []string{
			strconv.Itoa(int(opts.MaxItems)),
		}
	}
	f.Fields = append(f.Fields, field)

	// item_expire
	field = xep0004.Field{
		Var:   pubSubFormKey(itemExpireFormKey),
		Type:  xep0004.TextSingle,
		Label: "Number of seconds after which to automatically purge items. `max` for no specific limit other than a server imposed maximum.",
	}
	if opts.ItemExpire > 0 {
		field.Values = []string{
			strconv.Itoa(int(opts.ItemExpire)),
		}
	}
	f.Fields = append(f.Fields, field)

	// subscribe
	field = xep0004.Field{
		Var:   pubSubFormKey(subscribeFormKey),
		Type:  xep0004.Boolean,
		Label: "Whether to allow subscriptions",
		Values: []string{
			strconv.FormatBool(opts.Subscribe),
		},
	}
	f.Fields = append(f.Fields, field)

	// access_model
	field = xep0004.Field{
		Var:   pubSubFormKey(accessModelFormKey),
		Type:  xep0004.ListSingle,
		Label: "Who may subscribe and retrieve items",
		Values: []string{
			opts.AccessModel,
		},
		Options: []xep0004.Option{
			{Value: "authorize"},
			{Value: "open"},
			{Value: "presence"},
			{Value: "roster"},
			{Value: "whitelist"},
		},
	}
	f.Fields = append(f.Fields, field)

	// roster_allowed_groups
	field = xep0004.Field{
		Var:    pubSubFormKey(rosterGroupsAllowedFormKey),
		Type:   xep0004.ListMulti,
		Label:  "The roster group(s) allowed to subscribe and retrieve items",
		Values: opts.RosterGroupsAllowed,
	}
	f.Fields = append(f.Fields, field)

	// publish_model
	field = xep0004.Field{
		Var:   pubSubFormKey(publishModelFormKey),
		Type:  xep0004.ListSingle,
		Label: "The publisher model",
		Values: []string{
			opts.PublishModel,
		},
		Options: []xep0004.Option{
			{Value: "publishers"},
			{Value: "subscribers"},
			{Value: "open"},
		},
	}
	f.Fields = append(f.Fields, field)

	// purge_offline
	field = xep0004.Field{
		Var:   pubSubFormKey(purgeOffLineFormKey),
		Type:  xep0004.Boolean,
		Label: "Whether to purge all items when the relevant publisher goes offline",
		Values: []string{
			strconv.FormatBool(opts.PurgeOffline),
		},
	}
	f.Fields = append(f.Fields, field)

	// max_payload_size
	field = xep0004.Field{
		Var:   pubSubFormKey(maxPayloadSizeFormKey),
		Type:  xep0004.TextSingle,
		Label: "The maximum payload size in bytes",
	}
	if opts.MaxPayloadSize > 0 {
		field.Values = []string{
			strconv.Itoa(int(opts.MaxPayloadSize)),
		}
	}
	f.Fields = append(f.Fields, field)

	// send_last_published_item
	field = xep0004.Field{
		Var:   pubSubFormKey(sendLastPublishedItemFormKey),
		Type:  xep0004.ListSingle,
		Label: "When to send the last published item",
		Values: []string{
			opts.SendLastPublishedItem,
		},
		Options: []xep0004.Option{
			{Value: "never", Label: "Never"},
			{Value: "on_sub", Label: "When a new subscription is processed"},
			{Value: "on_sub_and_presence", Label: "When a new subscription is processed and whenever a subscriber comes online"},
		},
	}
	f.Fields = append(f.Fields, field)

	// presence_based_delivery
	field = xep0004.Field{
		Var:   pubSubFormKey(presenceBasedDeliveryFormKey),
		Type:  xep0004.Boolean,
		Label: "Whether to deliver notifications to available users only",
		Values: []string{
			strconv.FormatBool(opts.PresenceBasedDelivery),
		},
	}
	f.Fields = append(f.Fields, field)

	// notification_type
	field = xep0004.Field{
		Var:   pubSubFormKey(notificationTypeFormKey),
		Type:  xep0004.ListSingle,
		Label: "Specify the delivery style for notifications",
		Values: []string{
			opts.NotificationType,
		},
		Options: []xep0004.Option{
			{Value: "normal"},
			{Value: "headline"},
		},
	}
	f.Fields = append(f.Fields, field)

	// type
	field = xep0004.Field{
		Var:   pubSubFormKey(typeFormKey),
		Type:  xep0004.TextSingle,
		Label: "The semantic type information of data in the node, usually specified by the namespace of the payload (if any)",
	}
	if len(opts.Type) > 0 {
		field.Values = []string{
			opts.Type,
		}
	}
	f.Fields = append(f.Fields, field)

	// body_xslt
	field = xep0004.Field{
		Var:   pubSubFormKey(bodyXSLTFormKey),
		Type:  xep0004.TextSingle,
		Label: "The URL of an XSL transformation which can be applied to payloads in order to generate an appropriate message body element.",
	}
	if len(opts.BodyXslt) > 0 {
		field.Values = []string{
			opts.BodyXslt,
		}
	}
	f.Fields = append(f.Fields, field)

	// dataform_xslt
	field = xep0004.Field{
		Var:   pubSubFormKey(dataFormXSLTFormKey),
		Type:  xep0004.TextSingle,
		Label: "The URL of an XSL transformation which can be applied to the payload format in order to generate a valid Data Forms result that the client could display using a generic Data Forms rendering engine",
	}
	if len(opts.DataformXslt) > 0 {
		field.Values = []string{
			opts.DataformXslt,
		}
	}
	f.Fields = append(f.Fields, field)

	return &f
}

func pubSubFormKey(key string) string {
	return "pubsub#" + key
}
