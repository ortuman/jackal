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
	"testing"

	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"
	"github.com/ortuman/jackal/pkg/module/xep0004"
	"github.com/stretchr/testify/require"
)

func TestFormToOptions(t *testing.T) {
	x := &xep0004.DataForm{
		Type: xep0004.Submit,
		Fields: xep0004.Fields{
			{
				Var:    xep0004.FormType,
				Type:   xep0004.Hidden,
				Values: []string{pubSubNodeConfigNS()},
			},
			{
				Var:    pubSubFormKey(languageFormKey),
				Values: []string{"en"},
			},
			{
				Var:    pubSubFormKey(titleFormKey),
				Values: []string{"a title"},
			},
			{
				Var:    pubSubFormKey(descriptionFormKey),
				Values: []string{"a description"},
			},
			{
				Var:    pubSubFormKey(deliverPayloadsFormKey),
				Values: []string{"true"},
			},
			{
				Var:    pubSubFormKey(deliverNotificationsFormKey),
				Values: []string{"true"},
			},
			{
				Var:    pubSubFormKey(notifyConfigFormKey),
				Values: []string{"true"},
			},
			{
				Var:    pubSubFormKey(notifyDeleteFormKey),
				Values: []string{"true"},
			},
			{
				Var:    pubSubFormKey(notifyRetractFormKey),
				Values: []string{"true"},
			},
			{
				Var:    pubSubFormKey(notifySubFormKey),
				Values: []string{"true"},
			},
			{
				Var:    pubSubFormKey(persistItemsFormKey),
				Values: []string{"true"},
			},
			{
				Var:    pubSubFormKey(maxItemsFormKey),
				Values: []string{"1024"},
			},
			{
				Var:    pubSubFormKey(itemExpireFormKey),
				Values: []string{"86400"},
			},
			{
				Var:    pubSubFormKey(subscribeFormKey),
				Values: []string{"true"},
			},
			{
				Var:    pubSubFormKey(accessModelFormKey),
				Values: []string{"open"},
			},
			{
				Var:    pubSubFormKey(rosterGroupsAllowedFormKey),
				Values: []string{"friends", "family"},
			},
			{
				Var:    pubSubFormKey(publishModelFormKey),
				Values: []string{"publishers"},
			},
			{
				Var:    pubSubFormKey(purgeOffLineFormKey),
				Values: []string{"true"},
			},
			{
				Var:    pubSubFormKey(maxPayloadSizeFormKey),
				Values: []string{"1048576"},
			},
			{
				Var:    pubSubFormKey(sendLastPublishedItemFormKey),
				Values: []string{"on_sub_and_presence"},
			},
			{
				Var:    pubSubFormKey(presenceBasedDeliveryFormKey),
				Values: []string{"true"},
			},
			{
				Var:    pubSubFormKey(notificationTypeFormKey),
				Values: []string{"headline"},
			},
			{
				Var:    pubSubFormKey(typeFormKey),
				Values: []string{"a type"},
			},
			{
				Var:    pubSubFormKey(bodyXSLTFormKey),
				Values: []string{"a body xslt"},
			},
			{
				Var:    pubSubFormKey(dataFormXSLTFormKey),
				Values: []string{"a dataform xslt"},
			},
		},
	}

	opts, err := formToOptions(&pubsubmodel.Options{}, x.Element())
	require.NoError(t, err)

	require.Equal(t, "en", opts.Language)
	require.Equal(t, "a title", opts.Title)
	require.Equal(t, "a description", opts.Description)
	require.True(t, opts.DeliverPayloads)
	require.True(t, opts.DeliverNotifications)
	require.True(t, opts.NotifyConfig)
	require.True(t, opts.NotifyDelete)
	require.True(t, opts.NotifyRetract)
	require.True(t, opts.NotifySub)
	require.True(t, opts.PersistItems)
	require.Equal(t, int64(1024), opts.MaxItems)
	require.Equal(t, uint64(86400), opts.ItemExpire)
	require.True(t, opts.Subscribe)
	require.Equal(t, "open", opts.AccessModel)
	require.Equal(t, []string{"friends", "family"}, opts.RosterGroupsAllowed)
	require.Equal(t, "publishers", opts.PublishModel)
	require.True(t, opts.PurgeOffline)
	require.Equal(t, int64(1048576), opts.MaxPayloadSize)
	require.Equal(t, "on_sub_and_presence", opts.SendLastPublishedItem)
	require.True(t, opts.PresenceBasedDelivery)
	require.Equal(t, "headline", opts.NotificationType)
	require.Equal(t, "a type", opts.Type)
	require.Equal(t, "a body xslt", opts.BodyXslt)
	require.Equal(t, "a dataform xslt", opts.DataformXslt)
}

func TestOptionsToForm(t *testing.T) {
	// given
	opts := &pubsubmodel.Options{
		Language:              "en",
		Title:                 "a title",
		Description:           "a description",
		DeliverPayloads:       true,
		DeliverNotifications:  false,
		NotifyConfig:          true,
		NotifyDelete:          true,
		NotifyRetract:         true,
		NotifySub:             true,
		PersistItems:          true,
		MaxItems:              128,
		ItemExpire:            86400,
		Subscribe:             true,
		AccessModel:           "open",
		RosterGroupsAllowed:   []string{"friends", "family"},
		PublishModel:          "publishers",
		PurgeOffline:          false,
		MaxPayloadSize:        1048576,
		SendLastPublishedItem: "on_sub_and_presence",
		PresenceBasedDelivery: true,
		NotificationType:      "headline",
		Type:                  "a type",
		BodyXslt:              "a body xslt",
		DataformXslt:          "a dataform xslt",
	}

	// when
	f := optionsToForm(opts, xep0004.Result)

	// then
	require.NotNil(t, f)

	require.Equal(t, xep0004.FormType, f.Fields[0].Var)
	require.Equal(t, xep0004.Hidden, f.Fields[0].Type)

	// language
	require.Equal(t, pubSubFormKey(languageFormKey), f.Fields[1].Var)
	require.Equal(t, xep0004.TextSingle, f.Fields[1].Type)
	require.Equal(t, "en", f.Fields[1].Values[0])
	require.True(t, len(f.Fields[1].Label) > 0)

	// title
	require.Equal(t, pubSubFormKey(titleFormKey), f.Fields[2].Var)
	require.Equal(t, xep0004.TextSingle, f.Fields[2].Type)
	require.Equal(t, "a title", f.Fields[2].Values[0])
	require.True(t, len(f.Fields[2].Label) > 0)

	// description
	require.Equal(t, pubSubFormKey(descriptionFormKey), f.Fields[3].Var)
	require.Equal(t, xep0004.TextSingle, f.Fields[3].Type)
	require.Equal(t, "a description", f.Fields[3].Values[0])
	require.True(t, len(f.Fields[3].Label) > 0)

	// deliver_payloads
	require.Equal(t, pubSubFormKey(deliverPayloadsFormKey), f.Fields[4].Var)
	require.Equal(t, xep0004.Boolean, f.Fields[4].Type)
	require.Equal(t, "true", f.Fields[4].Values[0])
	require.True(t, len(f.Fields[4].Label) > 0)

	// deliver_notifications
	require.Equal(t, pubSubFormKey(deliverNotificationsFormKey), f.Fields[5].Var)
	require.Equal(t, xep0004.Boolean, f.Fields[5].Type)
	require.Equal(t, "false", f.Fields[5].Values[0])
	require.True(t, len(f.Fields[5].Label) > 0)

	// notify_config
	require.Equal(t, pubSubFormKey(notifyConfigFormKey), f.Fields[6].Var)
	require.Equal(t, xep0004.Boolean, f.Fields[6].Type)
	require.Equal(t, "true", f.Fields[6].Values[0])
	require.True(t, len(f.Fields[6].Label) > 0)

	// notify_delete
	require.Equal(t, pubSubFormKey(notifyDeleteFormKey), f.Fields[7].Var)
	require.Equal(t, xep0004.Boolean, f.Fields[7].Type)
	require.Equal(t, "true", f.Fields[7].Values[0])
	require.True(t, len(f.Fields[7].Label) > 0)

	// notify_retract
	require.Equal(t, pubSubFormKey(notifyRetractFormKey), f.Fields[8].Var)
	require.Equal(t, xep0004.Boolean, f.Fields[8].Type)
	require.Equal(t, "true", f.Fields[8].Values[0])
	require.True(t, len(f.Fields[8].Label) > 0)

	// notify_sub
	require.Equal(t, pubSubFormKey(notifySubFormKey), f.Fields[9].Var)
	require.Equal(t, xep0004.Boolean, f.Fields[9].Type)
	require.Equal(t, "true", f.Fields[9].Values[0])
	require.True(t, len(f.Fields[9].Label) > 0)

	// persist_items
	require.Equal(t, pubSubFormKey(persistItemsFormKey), f.Fields[10].Var)
	require.Equal(t, xep0004.Boolean, f.Fields[10].Type)
	require.Equal(t, "true", f.Fields[10].Values[0])
	require.True(t, len(f.Fields[10].Label) > 0)

	// max_items
	require.Equal(t, pubSubFormKey(maxItemsFormKey), f.Fields[11].Var)
	require.Equal(t, xep0004.TextSingle, f.Fields[11].Type)
	require.Equal(t, "128", f.Fields[11].Values[0])
	require.True(t, len(f.Fields[11].Label) > 0)

	// item_expire
	require.Equal(t, pubSubFormKey(itemExpireFormKey), f.Fields[12].Var)
	require.Equal(t, xep0004.TextSingle, f.Fields[12].Type)
	require.Equal(t, "86400", f.Fields[12].Values[0])
	require.True(t, len(f.Fields[12].Label) > 0)

	// subscribe
	require.Equal(t, pubSubFormKey(subscribeFormKey), f.Fields[13].Var)
	require.Equal(t, xep0004.Boolean, f.Fields[13].Type)
	require.Equal(t, "true", f.Fields[13].Values[0])
	require.True(t, len(f.Fields[13].Label) > 0)

	// access_model
	require.Equal(t, pubSubFormKey(accessModelFormKey), f.Fields[14].Var)
	require.Equal(t, xep0004.ListSingle, f.Fields[14].Type)
	require.Equal(t, "open", f.Fields[14].Values[0])
	require.True(t, len(f.Fields[14].Label) > 0)

	// roster_groups_allowed
	require.Equal(t, pubSubFormKey(rosterGroupsAllowedFormKey), f.Fields[15].Var)
	require.Equal(t, xep0004.ListMulti, f.Fields[15].Type)
	require.Equal(t, "friends", f.Fields[15].Values[0])
	require.Equal(t, "family", f.Fields[15].Values[1])
	require.True(t, len(f.Fields[15].Label) > 0)

	// publish_model
	require.Equal(t, pubSubFormKey(publishModelFormKey), f.Fields[16].Var)
	require.Equal(t, xep0004.ListSingle, f.Fields[16].Type)
	require.Equal(t, "publishers", f.Fields[16].Values[0])
	require.True(t, len(f.Fields[16].Label) > 0)

	// purge_offline
	require.Equal(t, pubSubFormKey(purgeOffLineFormKey), f.Fields[17].Var)
	require.Equal(t, xep0004.Boolean, f.Fields[17].Type)
	require.Equal(t, "false", f.Fields[17].Values[0])
	require.True(t, len(f.Fields[17].Label) > 0)

	// max_payload_size
	require.Equal(t, pubSubFormKey(maxPayloadSizeFormKey), f.Fields[18].Var)
	require.Equal(t, xep0004.TextSingle, f.Fields[18].Type)
	require.Equal(t, "1048576", f.Fields[18].Values[0])
	require.True(t, len(f.Fields[18].Label) > 0)

	// send_last_published_item
	require.Equal(t, pubSubFormKey(sendLastPublishedItemFormKey), f.Fields[19].Var)
	require.Equal(t, xep0004.ListSingle, f.Fields[19].Type)
	require.Equal(t, "on_sub_and_presence", f.Fields[19].Values[0])
	require.True(t, len(f.Fields[19].Label) > 0)

	// presence_based_delivery
	require.Equal(t, pubSubFormKey(presenceBasedDeliveryFormKey), f.Fields[20].Var)
	require.Equal(t, xep0004.Boolean, f.Fields[20].Type)
	require.Equal(t, "true", f.Fields[20].Values[0])
	require.True(t, len(f.Fields[20].Label) > 0)

	// notification_type
	require.Equal(t, pubSubFormKey(notificationTypeFormKey), f.Fields[21].Var)
	require.Equal(t, xep0004.ListSingle, f.Fields[21].Type)
	require.Equal(t, "headline", f.Fields[21].Values[0])
	require.True(t, len(f.Fields[21].Label) > 0)

	// type
	require.Equal(t, pubSubFormKey(typeFormKey), f.Fields[22].Var)
	require.Equal(t, xep0004.TextSingle, f.Fields[22].Type)
	require.Equal(t, "a type", f.Fields[22].Values[0])
	require.True(t, len(f.Fields[22].Label) > 0)

	// body_xslt
	require.Equal(t, pubSubFormKey(bodyXSLTFormKey), f.Fields[23].Var)
	require.Equal(t, xep0004.TextSingle, f.Fields[23].Type)
	require.Equal(t, "a body xslt", f.Fields[23].Values[0])
	require.True(t, len(f.Fields[23].Label) > 0)

	// dataform_xslt
	require.Equal(t, pubSubFormKey(dataFormXSLTFormKey), f.Fields[24].Var)
	require.Equal(t, xep0004.TextSingle, f.Fields[24].Type)
	require.Equal(t, "a dataform xslt", f.Fields[24].Values[0])
	require.True(t, len(f.Fields[24].Label) > 0)
}
