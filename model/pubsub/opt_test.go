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
	opt, err := NewOptionsFromSubmitForm(&xep0004.DataForm{})
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
				Var:    titleFieldVar,
				Values: []string{"Princely Musings (Atom)"},
			},
			{
				Var:    deliverNotificationsFieldVar,
				Values: []string{"1"},
			},
			{
				Var:    deliverPayloadsFieldVar,
				Values: []string{"1"},
			},
			{
				Var:    persistItemsFieldVar,
				Values: []string{"1"},
			},
			{
				Var:    maxItemsFieldVar,
				Values: []string{"10"},
			},
			{
				Var:    accessModelFieldVar,
				Values: []string{"open"},
			},
			{
				Var:    sendLastPublishedItemFieldVar,
				Values: []string{"never"},
			},
			{
				Var:    notificationTypeFieldVar,
				Values: []string{"headline"},
			},
			{
				Var:    notifyConfigFieldVar,
				Values: []string{"1"},
			},
			{
				Var:    notifyDeleteFieldVar,
				Values: []string{"TRUE"},
			},
			{
				Var:    notifyRetractFieldVar,
				Values: []string{"TRUE"},
			},
			{
				Var:    notifySubFieldVar,
				Values: []string{"TRUE"},
			},
		},
	}
	opt, err = NewOptionsFromSubmitForm(form)
	require.NotNil(t, opt)
	require.Nil(t, err)

	require.Equal(t, "Princely Musings (Atom)", opt.Title)
	require.True(t, opt.DeliverNotifications)
	require.True(t, opt.DeliverPayloads)
	require.True(t, opt.PersistItems)
	require.Equal(t, int64(10), opt.MaxItems)
	require.Equal(t, Open, opt.AccessModel)
	require.Equal(t, Never, opt.SendLastPublishedItem)
	require.Equal(t, "headline", opt.NotificationType)
	require.True(t, opt.NotifyConfig)
	require.True(t, opt.NotifyDelete)
	require.True(t, opt.NotifySub)

	form2 := opt.ResultForm()
	form2.Type = xep0004.Submit

	opt2, err := NewOptionsFromSubmitForm(form2)
	require.NotNil(t, opt2)
	require.Nil(t, err)

	require.True(t, reflect.DeepEqual(&opt, &opt2))

	optMap, _ := opt2.Map()
	opt3, err := NewOptionsFromMap(optMap)
	require.Nil(t, err)
	require.True(t, reflect.DeepEqual(&opt, &opt3))
}
