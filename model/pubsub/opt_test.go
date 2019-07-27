/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pubsubmodel

import (
	"reflect"
	"testing"

	"github.com/ortuman/jackal/module/xep0004"
	"github.com/stretchr/testify/require"
)

func TestOptions_New(t *testing.T) {
	opt, err := NewOptionsFromForm(&xep0004.DataForm{})
	require.Nil(t, opt)
	require.NotNil(t, err)

	form := &xep0004.DataForm{
		Type: xep0004.Submit,
		Fields: xep0004.Fields{
			{
				Var:    "FORM_TYPE",
				Type:   xep0004.Hidden,
				Values: []string{nodeConfigNamespace},
			},
			{
				Var:    titleOptField,
				Values: []string{"Princely Musings (Atom)"},
			},
			{
				Var:    deliverNotificationsOptField,
				Values: []string{"1"},
			},
			{
				Var:    deliverPayloadsOptField,
				Values: []string{"1"},
			},
			{
				Var:    persistItemsOptField,
				Values: []string{"1"},
			},
			{
				Var:    maxItemsOptField,
				Values: []string{"10"},
			},
			{
				Var:    itemExpireOptField,
				Values: []string{"604800"},
			},
			{
				Var:    accessModelOptField,
				Values: []string{"open"},
			},
			{
				Var:    publishModelOptField,
				Values: []string{"publishers"},
			},
			{
				Var:    purgeOfflineOptField,
				Values: []string{"true"},
			},
			{
				Var:    sendLastPublishedItemOptField,
				Values: []string{"never"},
			},
			{
				Var:    presenceBasedDeliveryOptField,
				Values: []string{"true"},
			},
			{
				Var:    notificationTypeOptField,
				Values: []string{"headline"},
			},
			{
				Var:    notifyConfigOptField,
				Values: []string{"1"},
			},
			{
				Var:    notifyDeleteOptField,
				Values: []string{"TRUE"},
			},
			{
				Var:    notifyRetractOptField,
				Values: []string{"TRUE"},
			},
			{
				Var:    notifySubOptField,
				Values: []string{"TRUE"},
			},
			{
				Var:    maxPayloadSizeOptField,
				Values: []string{"1024"},
			},
			{
				Var:    typeOptField,
				Values: []string{"http://www.w3.org/2005/Atom"},
			},
			{
				Var:    bodyXSLTOptField,
				Values: []string{"http://jabxslt.jabberstudio.org/atom_body.xslt"},
			},
		},
	}
	opt, err = NewOptionsFromForm(form)
	require.NotNil(t, opt)
	require.Nil(t, err)

	require.Equal(t, "Princely Musings (Atom)", opt.Title)
	require.True(t, opt.DeliverNotifications)
	require.True(t, opt.DeliverPayloads)
	require.True(t, opt.PersistItems)
	require.Equal(t, int64(10), opt.MaxItems)
	require.Equal(t, int64(604800), opt.ItemExpire)
	require.Equal(t, Open, opt.AccessModel)
	require.Equal(t, Publishers, opt.PublishModel)
	require.True(t, opt.PurgeOffline)
	require.Equal(t, Never, opt.SendLastPublishedItem)
	require.True(t, opt.PresenceBasedDelivery)
	require.Equal(t, "headline", opt.NotificationType)
	require.True(t, opt.NotifyConfig)
	require.True(t, opt.NotifyDelete)
	require.True(t, opt.NotifyRetract)
	require.True(t, opt.NotifySub)
	require.Equal(t, int64(1024), opt.MaxPayloadSize)
	require.Equal(t, "http://www.w3.org/2005/Atom", opt.Type)
	require.Equal(t, "http://jabxslt.jabberstudio.org/atom_body.xslt", opt.BodyXSLT)

	opt2, err := NewOptionsFromForm(opt.Form())
	require.NotNil(t, opt2)
	require.Nil(t, err)

	require.True(t, reflect.DeepEqual(&opt, &opt2))

	opt3, err := NewOptionsFromMap(opt2.Map())
	require.Nil(t, err)
	require.True(t, reflect.DeepEqual(&opt, &opt3))
}
