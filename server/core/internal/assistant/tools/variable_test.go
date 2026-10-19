package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/guregu/null/v5"
	"github.com/oxynote/oxynote/server/core/internal/tag"
	"github.com/oxynote/oxynote/server/core/internal/variable"
	"github.com/oxynote/oxynote/server/core/pkg/errutil"
	"github.com/oxynote/oxynote/server/core/pkg/testutil"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// _testVariableGroupID is the custom group the stubbed tree holds.
	_testVariableGroupID = xid.New()

	// _testVariableID is the list variable the stubbed tree holds.
	_testVariableID = xid.New()

	// _testOptionID is the first option of that list variable.
	_testOptionID = xid.New()

	// _testOtherOptionID is the second option of that list variable.
	_testOtherOptionID = xid.New()

	// _builtinVariableGroupID is the built-in group the stubbed tree holds.
	_builtinVariableGroupID = xid.New()

	// _builtinVariableID is the built-in variable the stubbed tree holds.
	_builtinVariableID = xid.New()

	// _unknownVariableID names nothing in the stubbed tree.
	_unknownVariableID = xid.New()
)

// stubVariableTree returns the tree the stubbed variable manager holds:
// the built-in group with its time range, then a custom group with a
// list variable.
func stubVariableTree() variable.Tree {
	return variable.Tree{
		{
			ID:        _builtinVariableGroupID,
			GroupName: "Metrics",
			Color:     "#155dfc",
			Builtin:   true,
			Variables: []variable.Entry{{
				ID:           _builtinVariableID,
				GroupID:      _builtinVariableGroupID,
				Key:          "metric_range",
				Name:         "Metric time range",
				Type:         variable.TypeTimeRange,
				DefaultValue: null.StringFrom("last_24_hours"),
				Builtin:      true,
				Value:        null.StringFrom("last_7_days"),
				Options:      []variable.Option{},
			}},
		},
		{
			ID:        _testVariableGroupID,
			GroupName: "Environment",
			Color:     "#123456",
			Variables: []variable.Entry{{
				ID:           _testVariableID,
				GroupID:      _testVariableGroupID,
				Key:          "region",
				Name:         "Region",
				Description:  null.StringFrom("Cloud region"),
				Type:         variable.TypeList,
				DefaultValue: null.StringFrom(_testOptionID.String()),
				Options: []variable.Option{
					{ID: _testOptionID, VariableID: _testVariableID, Value: "eu-west-1"},
					{ID: _testOtherOptionID, VariableID: _testVariableID, Value: "us-east-1", SortIndex: 1},
				},
			}},
		},
	}
}

// stubVariableManager answers tree reads with the stubbed tree and
// accepts every write.
func stubVariableManager() *VariableManagerMock {
	return &VariableManagerMock{
		FetchTreeFunc: func(context.Context, string, string) (variable.Tree, error) {
			return stubVariableTree(), nil
		},
		CreateGroupFunc: func(context.Context, variable.GroupCreateInput, string, string) (*variable.Group, error) {
			return &variable.Group{ID: _testVariableGroupID}, nil
		},
		CreateVariableFunc: func(context.Context, variable.Input, string, string) (*variable.Variable, error) {
			return &variable.Variable{ID: _testVariableID}, nil
		},
	}
}

// failingTreeManager is a variable manager whose tree read fails.
func failingTreeManager() *VariableManagerMock {
	return &VariableManagerMock{
		FetchTreeFunc: func(context.Context, string, string) (variable.Tree, error) {
			return nil, assert.AnError
		},
	}
}

// variableDeps returns test wiring over the variable manager.
func variableDeps(man *VariableManagerMock) *Deps {
	d := testDeps(nil, nil, nil)
	d.variableMan = man

	return d
}

func Test_listVariablesArgs_Validate(t *testing.T) {
	t.Parallel()

	assert.NoError(t, listVariablesArgs{}.Validate())
}

func Test_listVariables_Info(t *testing.T) {
	t.Parallel()

	info := listVariables{}.Info()

	assert.Equal(t, Traits{}, info.Traits)
	assert.Equal(t, NameListVariables, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Empty(t, info.Properties)
	assert.Empty(t, info.Required)
}

func Test_listVariables_Execute(t *testing.T) {
	t.Parallel()

	cc := map[string]struct {
		Man    *VariableManagerMock
		Args   string
		Result string
		Err    error
	}{
		"Malformed arguments": {Man: stubVariableManager(), Args: `{`, Err: assert.AnError},
		"Error returned by VariableManager.FetchTree": {
			Man:  failingTreeManager(),
			Args: `{}`,
			Err:  assert.AnError,
		},
		"Successful listing": {
			Man:  stubVariableManager(),
			Args: `{}`,
			Result: `{"groups":[
				{"id":"` + _builtinVariableGroupID.String() + `","name":"Metrics","color":"blue","builtin":true,"variables":[
					{"id":"` + _builtinVariableID.String() + `","key":"metric_range","name":"Metric time range","description":null,"type":"time-range","builtin":true,"default":"last_24_hours","value":"last_7_days","options":[]}
				]},
				{"id":"` + _testVariableGroupID.String() + `","name":"Environment","color":"unknown","builtin":false,"variables":[
					{"id":"` + _testVariableID.String() + `","key":"region","name":"Region","description":"Cloud region","type":"list","builtin":false,"default":"` + _testOptionID.String() + `","value":null,"options":[
						{"id":"` + _testOptionID.String() + `","value":"eu-west-1"},
						{"id":"` + _testOtherOptionID.String() + `","value":"us-east-1"}
					]}
				]}
			]}`,
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			res, err := listVariables{}.Execute(testInput(variableDeps(c.Man), NameListVariables, c.Args))
			testutil.AssertEqualError(t, c.Err, err)

			if err != nil {
				return
			}

			assert.JSONEq(t, c.Result, res)
		})
	}
}

func Test_createVariableGroupArgs_Validate(t *testing.T) {
	t.Parallel()

	assertValidate(t, createVariableGroupArgs{Name: "Deploys", ColorName: "orange"}, map[string]Args{
		"name":  createVariableGroupArgs{ColorName: "orange"},
		"color": createVariableGroupArgs{Name: "Deploys"},
	})

	assert.Equal(t, variable.ErrInvalidGroupColor, createVariableGroupArgs{Name: "Deploys", ColorName: "beige"}.Validate())
}

func Test_createVariableGroupArgs_input(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		variable.GroupCreateInput{GroupName: "Deploys", Color: "#f54a00"},
		createVariableGroupArgs{Name: "Deploys", ColorName: "orange"}.input(),
	)
}

func Test_createVariableGroup_Info(t *testing.T) {
	t.Parallel()

	info := createVariableGroup{}.Info()

	assert.Equal(t, Traits{Write: true}, info.Traits)
	assert.Equal(t, NameCreateVariableGroup, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Equal(t, []string{"name", "color"}, info.Required)
	assert.Equal(t, tag.ColorNames(), colorEnum(t, info))
}

func Test_createVariableGroup_Summary(t *testing.T) {
	t.Parallel()

	in := testInput(variableDeps(stubVariableManager()), NameCreateVariableGroup, `{`)

	_, err := createVariableGroup{}.Summary(in)
	assert.Error(t, err)

	in = testInput(variableDeps(stubVariableManager()), NameCreateVariableGroup, `{"name":"Deploys","color":"orange"}`)

	got, err := createVariableGroup{}.Summary(in)
	require.NoError(t, err)
	assert.Equal(t, ActionSummary{Tool: NameCreateVariableGroup, Summary: "Create variable group Deploys (orange)"}, got)
}

func Test_createVariableGroup_Execute(t *testing.T) {
	t.Parallel()

	cc := map[string]struct {
		Man  *VariableManagerMock
		Args string
		Err  error
	}{
		"Malformed arguments": {Man: stubVariableManager(), Args: `{`, Err: assert.AnError},
		"Error returned by VariableManager.CreateGroup": {
			Man: &VariableManagerMock{
				CreateGroupFunc: func(context.Context, variable.GroupCreateInput, string, string) (*variable.Group, error) {
					return nil, assert.AnError
				},
			},
			Args: `{"name":"Deploys","color":"orange"}`,
			Err:  assert.AnError,
		},
		"Created": {
			Man:  stubVariableManager(),
			Args: `{"name":"Deploys","color":"orange"}`,
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			res, err := createVariableGroup{}.Execute(testInput(variableDeps(c.Man), NameCreateVariableGroup, c.Args))
			testutil.AssertEqualError(t, c.Err, err)

			if err != nil {
				return
			}

			ff := c.Man.CreateGroupCalls()
			require.Len(t, ff, 1)
			assert.Equal(t, variable.GroupCreateInput{GroupName: "Deploys", Color: "#f54a00"}, ff[0].Inp)
			assert.Equal(t, "org", ff[0].OrganizationID)
			assert.Equal(t, "user", ff[0].UserID)
			assert.JSONEq(t, `{"group_id":"`+_testVariableGroupID.String()+`"}`, res)
		})
	}
}

func Test_updateVariableGroupArgs_Validate(t *testing.T) {
	t.Parallel()

	assertValidate(t, updateVariableGroupArgs{GroupID: _testVariableGroupID, Name: "Env"}, map[string]Args{
		"group_id": updateVariableGroupArgs{Name: "Env"},
	})

	assert.EqualError(t, updateVariableGroupArgs{GroupID: _testVariableGroupID}.Validate(), "give at least one of name or color")
	assert.Equal(t, variable.ErrInvalidGroupColor, updateVariableGroupArgs{GroupID: _testVariableGroupID, ColorName: "beige"}.Validate())
}

func Test_updateVariableGroupArgs_input(t *testing.T) {
	t.Parallel()

	assert.Equal(t, variable.GroupUpdateInput{GroupName: null.StringFrom("Env")}, updateVariableGroupArgs{Name: "Env"}.input())
	assert.Equal(t, variable.GroupUpdateInput{Color: null.StringFrom("#f54a00")}, updateVariableGroupArgs{ColorName: "orange"}.input())
}

func Test_updateVariableGroup_Info(t *testing.T) {
	t.Parallel()

	info := updateVariableGroup{}.Info()

	assert.Equal(t, Traits{Write: true}, info.Traits)
	assert.Equal(t, NameUpdateVariableGroup, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Equal(t, []string{"group_id"}, info.Required)
	assert.Contains(t, info.Properties, "name")
	assert.Contains(t, info.Properties, "color")
}

func Test_updateVariableGroup_Summary(t *testing.T) {
	t.Parallel()

	group := `"group_id":"` + _testVariableGroupID.String() + `"`

	cc := map[string]struct {
		Man    *VariableManagerMock
		Args   string
		Result string
		Err    error
	}{
		"Malformed arguments": {Man: stubVariableManager(), Args: `{`, Err: assert.AnError},
		"Error returned by VariableManager.FetchTree": {
			Man:  failingTreeManager(),
			Args: `{` + group + `,"name":"Env"}`,
			Err:  assert.AnError,
		},
		"Unknown group": {
			Man:  stubVariableManager(),
			Args: `{"group_id":"` + _unknownVariableID.String() + `","name":"Env"}`,
			Err:  assert.AnError,
		},
		"Rename and recolour": {
			Man:    stubVariableManager(),
			Args:   `{` + group + `,"name":"Env","color":"orange"}`,
			Result: `Rename variable group Environment to "Env" and set the colour of variable group Environment to orange`,
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			got, err := updateVariableGroup{}.Summary(testInput(variableDeps(c.Man), NameUpdateVariableGroup, c.Args))
			testutil.AssertEqualError(t, c.Err, err)

			if err != nil {
				return
			}

			assert.Equal(t, ActionSummary{Tool: NameUpdateVariableGroup, Summary: c.Result}, got)
		})
	}
}

func Test_updateVariableGroup_Execute(t *testing.T) {
	t.Parallel()

	cc := map[string]struct {
		Man  *VariableManagerMock
		Args string
		Err  error
	}{
		"Malformed arguments": {Man: stubVariableManager(), Args: `{`, Err: assert.AnError},
		"Unknown group": {
			Man: &VariableManagerMock{
				UpdateGroupFunc: func(context.Context, xid.ID, variable.GroupUpdateInput, string) error {
					return errutil.ErrNotFound
				},
			},
			Args: `{"group_id":"` + _testVariableGroupID.String() + `","name":"Env"}`,
			Err:  fmt.Errorf("variable group %s: %w", _testVariableGroupID, ErrUnknownVariableGroup),
		},
		"Updated": {
			Man:  stubVariableManager(),
			Args: `{"group_id":"` + _testVariableGroupID.String() + `","name":"Env"}`,
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			res, err := updateVariableGroup{}.Execute(testInput(variableDeps(c.Man), NameUpdateVariableGroup, c.Args))
			testutil.AssertEqualError(t, c.Err, err)

			if err != nil {
				return
			}

			ff := c.Man.UpdateGroupCalls()
			require.Len(t, ff, 1)
			assert.Equal(t, _testVariableGroupID, ff[0].ID)
			assert.Equal(t, variable.GroupUpdateInput{GroupName: null.StringFrom("Env")}, ff[0].Inp)
			assert.JSONEq(t, `{"group_id":"`+_testVariableGroupID.String()+`"}`, res)
		})
	}
}

func Test_deleteVariableGroupArgs_Validate(t *testing.T) {
	t.Parallel()

	assertValidate(t, deleteVariableGroupArgs{GroupID: _testVariableGroupID}, map[string]Args{
		"group_id": deleteVariableGroupArgs{},
	})
}

func Test_deleteVariableGroup_Info(t *testing.T) {
	t.Parallel()

	info := deleteVariableGroup{}.Info()

	assert.Equal(t, Traits{Write: true, Destructive: true}, info.Traits)
	assert.Equal(t, NameDeleteVariableGroup, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Equal(t, []string{"group_id"}, info.Required)
}

func Test_deleteVariableGroup_Summary(t *testing.T) {
	t.Parallel()

	cc := map[string]struct {
		Args   string
		Result string
		Err    error
	}{
		"Malformed arguments": {Args: `{`, Err: assert.AnError},
		"Unknown group": {
			Args: `{"group_id":"` + _unknownVariableID.String() + `"}`,
			Err:  assert.AnError,
		},
		"Known group": {
			Args:   `{"group_id":"` + _testVariableGroupID.String() + `"}`,
			Result: "Delete variable group Environment",
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			got, err := deleteVariableGroup{}.Summary(testInput(variableDeps(stubVariableManager()), NameDeleteVariableGroup, c.Args))
			testutil.AssertEqualError(t, c.Err, err)

			if err != nil {
				return
			}

			assert.Equal(t, ActionSummary{Tool: NameDeleteVariableGroup, Summary: c.Result}, got)
		})
	}
}

func Test_deleteVariableGroup_Execute(t *testing.T) {
	t.Parallel()

	cc := map[string]struct {
		Man  *VariableManagerMock
		Args string
		Err  error
	}{
		"Malformed arguments": {Man: stubVariableManager(), Args: `{`, Err: assert.AnError},
		"Group with variables": {
			Man: &VariableManagerMock{
				DeleteGroupFunc: func(context.Context, xid.ID, string) error {
					return variable.ErrGroupNotEmpty
				},
			},
			Args: `{"group_id":"` + _testVariableGroupID.String() + `"}`,
			Err:  variable.ErrGroupNotEmpty,
		},
		"Deleted": {
			Man:  stubVariableManager(),
			Args: `{"group_id":"` + _testVariableGroupID.String() + `"}`,
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			res, err := deleteVariableGroup{}.Execute(testInput(variableDeps(c.Man), NameDeleteVariableGroup, c.Args))
			testutil.AssertEqualError(t, c.Err, err)

			if err != nil {
				return
			}

			ff := c.Man.DeleteGroupCalls()
			require.Len(t, ff, 1)
			assert.Equal(t, _testVariableGroupID, ff[0].ID)
			assert.JSONEq(t, `{"group_id":"`+_testVariableGroupID.String()+`","deleted":true}`, res)
		})
	}
}

func Test_createVariableArgs_Validate(t *testing.T) {
	t.Parallel()

	ok := createVariableArgs{GroupID: _testVariableGroupID, Key: "release", Name: "Release", Type: variable.TypeText}

	assertValidate(t, ok, map[string]Args{
		"group_id": createVariableArgs{Key: "release", Name: "Release", Type: variable.TypeText},
		"key":      createVariableArgs{GroupID: _testVariableGroupID, Name: "Release", Type: variable.TypeText},
		"name":     createVariableArgs{GroupID: _testVariableGroupID, Key: "release", Type: variable.TypeText},
		"type":     createVariableArgs{GroupID: _testVariableGroupID, Key: "release", Name: "Release"},
	})

	bad := ok
	bad.Key = "$release"

	assert.Equal(t, variable.ErrInvalidKey, bad.Validate())
}

func Test_createVariableArgs_input(t *testing.T) {
	t.Parallel()

	got := createVariableArgs{
		GroupID:     _testVariableGroupID,
		Key:         "region",
		Name:        "Region",
		Description: "Cloud region",
		Type:        variable.TypeList,
		Options:     []string{"eu-west-1", "us-east-1"},
	}.input()

	assert.Equal(t, variable.Input{
		GroupID:     _testVariableGroupID,
		Key:         "region",
		Name:        "Region",
		Description: null.StringFrom("Cloud region"),
		Type:        variable.TypeList,
		Options:     []variable.OptionInput{{Value: "eu-west-1"}, {Value: "us-east-1"}},
	}, got)

	got = createVariableArgs{Type: variable.TypeNumber, Default: null.StringFrom("2")}.input()

	assert.False(t, got.Description.Valid)
	assert.Equal(t, null.StringFrom("2"), got.DefaultValue)
	assert.Nil(t, got.Options)
}

func Test_createVariable_Info(t *testing.T) {
	t.Parallel()

	info := createVariable{}.Info()

	assert.Equal(t, Traits{Write: true}, info.Traits)
	assert.Equal(t, NameCreateVariable, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Equal(t, []string{"group_id", "key", "name", "type"}, info.Required)
	assert.Contains(t, info.Properties, "options")
	assert.Contains(t, info.Properties, "default")
}

func Test_createVariable_Summary(t *testing.T) {
	t.Parallel()

	args := func(groupID xid.ID) string {
		return `{"group_id":"` + groupID.String() + `","key":"release","name":"Release","type":"text"}`
	}

	cc := map[string]struct {
		Args   string
		Result string
		Err    error
	}{
		"Malformed arguments": {Args: `{`, Err: assert.AnError},
		"Unknown group":       {Args: args(_unknownVariableID), Err: assert.AnError},
		"Known group": {
			Args:   args(_testVariableGroupID),
			Result: "Create text variable $release in Environment",
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			got, err := createVariable{}.Summary(testInput(variableDeps(stubVariableManager()), NameCreateVariable, c.Args))
			testutil.AssertEqualError(t, c.Err, err)

			if err != nil {
				return
			}

			assert.Equal(t, ActionSummary{Tool: NameCreateVariable, Summary: c.Result}, got)
		})
	}
}

func Test_createVariable_Execute(t *testing.T) {
	t.Parallel()

	args := `{"group_id":"` + _testVariableGroupID.String() + `","key":"region","name":"Region","type":"list","options":["eu-west-1"]}`

	cc := map[string]struct {
		Man  *VariableManagerMock
		Args string
		Err  error
	}{
		"Malformed arguments": {Man: stubVariableManager(), Args: `{`, Err: assert.AnError},
		"Unknown group": {
			Man: &VariableManagerMock{
				CreateVariableFunc: func(context.Context, variable.Input, string, string) (*variable.Variable, error) {
					return nil, errutil.ErrNotFound
				},
			},
			Args: args,
			Err:  fmt.Errorf("variable group %s: %w", _testVariableGroupID, ErrUnknownVariableGroup),
		},
		"Created": {
			Man:  stubVariableManager(),
			Args: args,
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			res, err := createVariable{}.Execute(testInput(variableDeps(c.Man), NameCreateVariable, c.Args))
			testutil.AssertEqualError(t, c.Err, err)

			if err != nil {
				return
			}

			ff := c.Man.CreateVariableCalls()
			require.Len(t, ff, 1)
			assert.Equal(t, "region", ff[0].Inp.Key)
			assert.Equal(t, []variable.OptionInput{{Value: "eu-west-1"}}, ff[0].Inp.Options)
			assert.Equal(t, "user", ff[0].UserID)
			assert.JSONEq(t, `{"variable_id":"`+_testVariableID.String()+`"}`, res)
		})
	}
}

func Test_updateVariableArgs_Validate(t *testing.T) {
	t.Parallel()

	assertValidate(t, updateVariableArgs{VariableID: _testVariableID}, map[string]Args{
		"variable_id": updateVariableArgs{},
	})

	assert.Equal(t, variable.ErrInvalidType, updateVariableArgs{VariableID: _testVariableID, Type: "date"}.Validate())
	assert.NoError(t, updateVariableArgs{VariableID: _testVariableID, Type: variable.TypeText}.Validate())
}

func Test_updateVariableArgs_input(t *testing.T) {
	t.Parallel()

	list := stubVariableTree()[1].Variables[0]
	builtin := stubVariableTree()[0].Variables[0]
	otherGroup := xid.New()

	cc := map[string]struct {
		Args    updateVariableArgs
		Current variable.Entry
		Result  variable.Input
	}{
		"Nothing given keeps a list as it is": {
			Current: list,
			Result: variable.Input{
				GroupID:     _testVariableGroupID,
				Key:         "region",
				Name:        "Region",
				Description: null.StringFrom("Cloud region"),
				Type:        variable.TypeList,
				Options: []variable.OptionInput{
					{ID: null.ValueFrom(_testOptionID), Value: "eu-west-1"},
					{ID: null.ValueFrom(_testOtherOptionID), Value: "us-east-1"},
				},
			},
		},
		"New options keep the ids of values the list has": {
			Args:    updateVariableArgs{GroupID: otherGroup, Key: "zone", Name: "Zone", Description: null.StringFrom(""), Options: []string{"us-east-1", "ap-south-1"}},
			Current: list,
			Result: variable.Input{
				GroupID: otherGroup,
				Key:     "zone",
				Name:    "Zone",
				Type:    variable.TypeList,
				Options: []variable.OptionInput{
					{ID: null.ValueFrom(_testOtherOptionID), Value: "us-east-1"},
					{Value: "ap-south-1"},
				},
			},
		},
		"Repeated values name one option once": {
			Args:    updateVariableArgs{Options: []string{"us-east-1", "us-east-1"}},
			Current: list,
			Result: variable.Input{
				GroupID:     _testVariableGroupID,
				Key:         "region",
				Name:        "Region",
				Description: null.StringFrom("Cloud region"),
				Type:        variable.TypeList,
				Options:     []variable.OptionInput{{ID: null.ValueFrom(_testOtherOptionID), Value: "us-east-1"}},
			},
		},
		"List turned into text drops its options": {
			Args:    updateVariableArgs{Type: variable.TypeText, Default: null.StringFrom("eu")},
			Current: list,
			Result: variable.Input{
				GroupID:      _testVariableGroupID,
				Key:          "region",
				Name:         "Region",
				Description:  null.StringFrom("Cloud region"),
				Type:         variable.TypeText,
				DefaultValue: null.StringFrom("eu"),
			},
		},
		"Same type keeps the default": {
			Args:    updateVariableArgs{Description: null.StringFrom("Window")},
			Current: builtin,
			Result: variable.Input{
				GroupID:      _builtinVariableGroupID,
				Key:          "metric_range",
				Name:         "Metric time range",
				Description:  null.StringFrom("Window"),
				Type:         variable.TypeTimeRange,
				DefaultValue: null.StringFrom("last_24_hours"),
			},
		},
		"Type change drops the old default": {
			Args:    updateVariableArgs{Type: variable.TypeText},
			Current: builtin,
			Result: variable.Input{
				GroupID: _builtinVariableGroupID,
				Key:     "metric_range",
				Name:    "Metric time range",
				Type:    variable.TypeText,
			},
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, c.Result, c.Args.input(c.Current))
		})
	}
}

func Test_updateVariable_Info(t *testing.T) {
	t.Parallel()

	info := updateVariable{}.Info()

	assert.Equal(t, Traits{Write: true}, info.Traits)
	assert.Equal(t, NameUpdateVariable, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Equal(t, []string{"variable_id"}, info.Required)

	// every optional argument says what it does, not only that it is
	// optional.
	for key, prop := range info.Properties {
		p, ok := prop.(map[string]any)
		require.True(t, ok)
		assert.NotEqual(t, "Optional.", p["description"], key)
	}
}

func Test_updateVariable_Summary(t *testing.T) {
	t.Parallel()

	cc := map[string]struct {
		Args   string
		Result string
		Err    error
	}{
		"Malformed arguments": {Args: `{`, Err: assert.AnError},
		"Unknown variable": {
			Args: `{"variable_id":"` + _unknownVariableID.String() + `"}`,
			Err:  assert.AnError,
		},
		"Same type": {
			Args:   `{"variable_id":"` + _testVariableID.String() + `","name":"Zone"}`,
			Result: "Update variable $region",
		},
		"Type change": {
			Args:   `{"variable_id":"` + _testVariableID.String() + `","type":"text"}`,
			Result: "Update variable $region and make it text, resetting everyone's value",
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			got, err := updateVariable{}.Summary(testInput(variableDeps(stubVariableManager()), NameUpdateVariable, c.Args))
			testutil.AssertEqualError(t, c.Err, err)

			if err != nil {
				return
			}

			assert.Equal(t, ActionSummary{Tool: NameUpdateVariable, Summary: c.Result}, got)
		})
	}
}

func Test_updateVariable_Execute(t *testing.T) {
	t.Parallel()

	args := `{"variable_id":"` + _testVariableID.String() + `","name":"Zone"}`

	withUpdate := func(err error) *VariableManagerMock {
		m := stubVariableManager()
		m.UpdateVariableFunc = func(context.Context, xid.ID, variable.Input, string) error {
			return err
		}

		return m
	}

	cc := map[string]struct {
		Man  *VariableManagerMock
		Args string
		Err  error
	}{
		"Malformed arguments": {Man: stubVariableManager(), Args: `{`, Err: assert.AnError},
		"Unknown variable": {
			Man:  stubVariableManager(),
			Args: `{"variable_id":"` + _unknownVariableID.String() + `"}`,
			Err:  fmt.Errorf("variable %s: %w", _unknownVariableID, ErrUnknownVariable),
		},
		"Built-in variable": {
			Man:  withUpdate(variable.ErrBuiltinVariable),
			Args: args,
			Err:  variable.ErrBuiltinVariable,
		},
		"Updated": {
			Man:  withUpdate(nil),
			Args: args,
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			res, err := updateVariable{}.Execute(testInput(variableDeps(c.Man), NameUpdateVariable, c.Args))
			testutil.AssertEqualError(t, c.Err, err)

			if err != nil {
				return
			}

			ff := c.Man.UpdateVariableCalls()
			require.Len(t, ff, 1)
			assert.Equal(t, _testVariableID, ff[0].ID)
			assert.Equal(t, "Zone", ff[0].Inp.Name)
			assert.Len(t, ff[0].Inp.Options, 2)
			assert.JSONEq(t, `{"variable_id":"`+_testVariableID.String()+`"}`, res)
		})
	}
}

func Test_deleteVariableArgs_Validate(t *testing.T) {
	t.Parallel()

	assertValidate(t, deleteVariableArgs{VariableID: _testVariableID}, map[string]Args{
		"variable_id": deleteVariableArgs{},
	})
}

func Test_deleteVariable_Info(t *testing.T) {
	t.Parallel()

	info := deleteVariable{}.Info()

	assert.Equal(t, Traits{Write: true, Destructive: true}, info.Traits)
	assert.Equal(t, NameDeleteVariable, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Equal(t, []string{"variable_id"}, info.Required)
}

func Test_deleteVariable_Summary(t *testing.T) {
	t.Parallel()

	cc := map[string]struct {
		Args   string
		Result string
		Err    error
	}{
		"Malformed arguments": {Args: `{`, Err: assert.AnError},
		"Unknown variable":    {Args: `{"variable_id":"` + _unknownVariableID.String() + `"}`, Err: assert.AnError},
		"Known variable": {
			Args:   `{"variable_id":"` + _testVariableID.String() + `"}`,
			Result: "Delete variable $region",
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			got, err := deleteVariable{}.Summary(testInput(variableDeps(stubVariableManager()), NameDeleteVariable, c.Args))
			testutil.AssertEqualError(t, c.Err, err)

			if err != nil {
				return
			}

			assert.Equal(t, ActionSummary{Tool: NameDeleteVariable, Summary: c.Result}, got)
		})
	}
}

func Test_deleteVariable_Execute(t *testing.T) {
	t.Parallel()

	cc := map[string]struct {
		Man  *VariableManagerMock
		Args string
		Err  error
	}{
		"Malformed arguments": {Man: stubVariableManager(), Args: `{`, Err: assert.AnError},
		"Unknown variable": {
			Man: &VariableManagerMock{
				DeleteVariableFunc: func(context.Context, xid.ID, string) error {
					return errutil.ErrNotFound
				},
			},
			Args: `{"variable_id":"` + _testVariableID.String() + `"}`,
			Err:  fmt.Errorf("variable %s: %w", _testVariableID, ErrUnknownVariable),
		},
		"Deleted": {
			Man:  stubVariableManager(),
			Args: `{"variable_id":"` + _testVariableID.String() + `"}`,
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			res, err := deleteVariable{}.Execute(testInput(variableDeps(c.Man), NameDeleteVariable, c.Args))
			testutil.AssertEqualError(t, c.Err, err)

			if err != nil {
				return
			}

			ff := c.Man.DeleteVariableCalls()
			require.Len(t, ff, 1)
			assert.Equal(t, _testVariableID, ff[0].ID)
			assert.JSONEq(t, `{"variable_id":"`+_testVariableID.String()+`","deleted":true}`, res)
		})
	}
}

func Test_setVariableValueArgs_Validate(t *testing.T) {
	t.Parallel()

	assertValidate(t, setVariableValueArgs{VariableID: _testVariableID, Value: null.StringFrom(""), Default: true}, map[string]Args{
		"variable_id": setVariableValueArgs{Value: null.StringFrom("x")},
		"value":       setVariableValueArgs{VariableID: _testVariableID},
	})
}

func Test_setVariableValue_Info(t *testing.T) {
	t.Parallel()

	info := setVariableValue{}.Info()

	assert.Equal(t, Traits{Write: true}, info.Traits)
	assert.Equal(t, NameSetVariableValue, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Equal(t, []string{"variable_id", "value"}, info.Required)
}

func Test_setVariableValue_Summary(t *testing.T) {
	t.Parallel()

	cc := map[string]struct {
		Args   string
		Result string
		Err    error
	}{
		"Malformed arguments": {Args: `{`, Err: assert.AnError},
		"Unknown variable": {
			Args: `{"variable_id":"` + _unknownVariableID.String() + `","value":"x"}`,
			Err:  assert.AnError,
		},
		"Option id shown as its value": {
			Args:   `{"variable_id":"` + _testVariableID.String() + `","value":"` + _testOtherOptionID.String() + `"}`,
			Result: `Set your $region to "us-east-1"`,
		},
		"Plain value": {
			Args:   `{"variable_id":"` + _builtinVariableID.String() + `","value":"last_1_hour"}`,
			Result: `Set your $metric_range to "last_1_hour"`,
		},
		"Default": {
			Args:   `{"variable_id":"` + _builtinVariableID.String() + `","value":"last_7_days","default":true}`,
			Result: `Set the default of $metric_range to "last_7_days"`,
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			got, err := setVariableValue{}.Summary(testInput(variableDeps(stubVariableManager()), NameSetVariableValue, c.Args))
			testutil.AssertEqualError(t, c.Err, err)

			if err != nil {
				return
			}

			assert.Equal(t, ActionSummary{Tool: NameSetVariableValue, Summary: c.Result}, got)
		})
	}
}

func Test_setVariableValue_Execute(t *testing.T) {
	t.Parallel()

	cc := map[string]struct {
		Man         *VariableManagerMock
		Args        string
		ValueCalls  int
		DefaultCall bool
		Result      string
		Err         error
	}{
		"Malformed arguments": {Man: stubVariableManager(), Args: `{`, Err: assert.AnError},
		"Invalid value": {
			Man: &VariableManagerMock{
				SetValueFunc: func(context.Context, xid.ID, string, string, string) error {
					return variable.ErrInvalidTimeRange
				},
			},
			Args: `{"variable_id":"` + _builtinVariableID.String() + `","value":"last_1_hour"}`,
			Err:  variable.ErrInvalidTimeRange,
		},
		"Invalid default": {
			Man: &VariableManagerMock{
				SetDefaultValueFunc: func(context.Context, xid.ID, string, string) error {
					return variable.ErrInvalidTimeRange
				},
			},
			Args: `{"variable_id":"` + _builtinVariableID.String() + `","value":"last_1_hour","default":true}`,
			Err:  variable.ErrInvalidTimeRange,
		},
		"Set": {
			Man:        stubVariableManager(),
			Args:       `{"variable_id":"` + _builtinVariableID.String() + `","value":"last_1_hour"}`,
			ValueCalls: 1,
			Result:     `{"variable_id":"` + _builtinVariableID.String() + `","value":"last_1_hour","default":false}`,
		},
		"Set the default": {
			Man:         stubVariableManager(),
			Args:        `{"variable_id":"` + _builtinVariableID.String() + `","value":"last_1_hour","default":true}`,
			DefaultCall: true,
			Result:      `{"variable_id":"` + _builtinVariableID.String() + `","value":"last_1_hour","default":true}`,
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			res, err := setVariableValue{}.Execute(testInput(variableDeps(c.Man), NameSetVariableValue, c.Args))
			testutil.AssertEqualError(t, c.Err, err)

			if err != nil {
				return
			}

			assert.JSONEq(t, c.Result, res)

			vv := c.Man.SetValueCalls()
			require.Len(t, vv, c.ValueCalls)

			for _, v := range vv {
				assert.Equal(t, _builtinVariableID, v.ID)
				assert.Equal(t, "last_1_hour", v.Value)
				assert.Equal(t, "org", v.OrganizationID)
				assert.Equal(t, "user", v.UserID)
			}

			dd := c.Man.SetDefaultValueCalls()

			if !c.DefaultCall {
				assert.Empty(t, dd)

				return
			}

			require.Len(t, dd, 1)
			assert.Equal(t, _builtinVariableID, dd[0].ID)
			assert.Equal(t, "last_1_hour", dd[0].Value)
			assert.Equal(t, "org", dd[0].OrganizationID)
		})
	}
}
