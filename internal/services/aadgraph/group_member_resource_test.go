package aadgraph_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/terraform-providers/terraform-provider-azuread/internal/acceptance"
	"github.com/terraform-providers/terraform-provider-azuread/internal/acceptance/check"
	"github.com/terraform-providers/terraform-provider-azuread/internal/clients"
	"github.com/terraform-providers/terraform-provider-azuread/internal/services/aadgraph/graph"
	"github.com/terraform-providers/terraform-provider-azuread/internal/utils"
)

type GroupMemberResource struct{}

func TestAccGroupMember_group(t *testing.T) {
	data := acceptance.BuildTestData(t, "azuread_group_member", "test")
	r := GroupMemberResource{}

	data.ResourceTest(t, r, []resource.TestStep{
		{
			Config: r.group(data),
			Check: resource.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("group_object_id").IsGuid(),
				check.That(data.ResourceName).Key("member_object_id").IsGuid(),
			),
		},
		data.ImportStep(),
	})
}

func TestAccGroupMember_servicePrincipal(t *testing.T) {
	data := acceptance.BuildTestData(t, "azuread_group_member", "test")
	r := GroupMemberResource{}

	data.ResourceTest(t, r, []resource.TestStep{
		{
			Config: r.servicePrincipal(data),
			Check: resource.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("group_object_id").IsGuid(),
				check.That(data.ResourceName).Key("member_object_id").IsGuid(),
			),
		},
		data.ImportStep(),
	})
}

func TestAccGroupMember_user(t *testing.T) {
	data := acceptance.BuildTestData(t, "azuread_group_member", "testA")
	r := GroupMemberResource{}

	data.ResourceTest(t, r, []resource.TestStep{
		{
			Config: r.oneUser(data),
			Check: resource.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("group_object_id").IsGuid(),
				check.That(data.ResourceName).Key("member_object_id").IsGuid(),
			),
		},
		data.ImportStep(),
	})
}

func TestAccGroupMember_multipleUser(t *testing.T) {
	dataA := acceptance.BuildTestData(t, "azuread_group_member", "testA")
	dataB := acceptance.BuildTestData(t, "azuread_group_member", "testB")
	r := GroupMemberResource{}

	dataA.ResourceTest(t, r, []resource.TestStep{
		{
			Config: r.oneUser(dataA),
			Check: resource.ComposeTestCheckFunc(
				check.That(dataA.ResourceName).ExistsInAzure(r),
				check.That(dataA.ResourceName).Key("group_object_id").IsGuid(),
				check.That(dataA.ResourceName).Key("member_object_id").IsGuid(),
			),
		},
		dataA.ImportStep(),
		{
			Config: r.twoUsers(dataA),
			Check: resource.ComposeTestCheckFunc(
				check.That(dataA.ResourceName).ExistsInAzure(r),
				check.That(dataA.ResourceName).Key("group_object_id").IsGuid(),
				check.That(dataA.ResourceName).Key("member_object_id").IsGuid(),
				check.That(dataB.ResourceName).ExistsInAzure(r),
				check.That(dataB.ResourceName).Key("group_object_id").IsGuid(),
				check.That(dataB.ResourceName).Key("member_object_id").IsGuid(),
			),
		},
		// we rerun the config so the group resource updates with the number of members
		{
			Config: r.twoUsers(dataA),
			Check: resource.ComposeTestCheckFunc(
				check.That("azuread_group.test").Key("members.#").HasValue("2"),
			),
		},
		dataA.ImportStep(),
		{
			Config: r.oneUser(dataA),
			Check: resource.ComposeTestCheckFunc(
				check.That(dataA.ResourceName).ExistsInAzure(r),
				check.That(dataA.ResourceName).Key("group_object_id").IsGuid(),
				check.That(dataA.ResourceName).Key("member_object_id").IsGuid(),
			),
		},
		// we rerun the config so the group resource updates with the number of members
		{
			Config: r.oneUser(dataA),
			Check: resource.ComposeTestCheckFunc(
				check.That("azuread_group.test").Key("members.#").HasValue("1"),
			),
		},
	})
}

func TestAccGroupMember_requiresImport(t *testing.T) {
	data := acceptance.BuildTestData(t, "azuread_group_member", "test")
	r := GroupMemberResource{}

	data.ResourceTest(t, r, []resource.TestStep{
		{
			Config: r.group(data),
			Check: resource.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.RequiresImportErrorStep(r.requiresImport(data)),
	})
}

func (r GroupMemberResource) Exists(ctx context.Context, clients *clients.AadClient, state *terraform.InstanceState) (*bool, error) {
	id, err := graph.ParseGroupMemberId(state.ID)
	if err != nil {
		return nil, fmt.Errorf("parsing Group Member ID: %v", err)
	}

	if resp, err := clients.AadGraph.GroupsClient.Get(ctx, id.GroupId); err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			return nil, fmt.Errorf("Group with object ID %q does not exist", id.GroupId)
		}

		return nil, fmt.Errorf("failed to retrieve Group with object ID %q: %+v", id.GroupId, err)
	}

	members, err := graph.GroupAllMembers(ctx, clients.AadGraph.GroupsClient, id.GroupId)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve Group members (groupId: %q): %+v", id.GroupId, err)
	}

	for _, memberId := range members {
		if memberId == id.MemberId {
			return utils.Bool(true), nil
		}
	}

	return nil, fmt.Errorf("Member %q was not found in Group %q", id.MemberId, id.GroupId)
}

func (GroupMemberResource) group(data acceptance.TestData) string {
	return fmt.Sprintf(`
%[1]s

resource "azuread_group" "member" {
  name = "acctestGroup-%[2]d-Member"
}

resource "azuread_group_member" "test" {
  group_object_id  = azuread_group.test.object_id
  member_object_id = azuread_group.member.object_id
}
`, GroupResource{}.basic(data), data.RandomInteger)
}

func (GroupMemberResource) servicePrincipal(data acceptance.TestData) string {
	return fmt.Sprintf(`
%[1]s
%[2]s

resource "azuread_group_member" "test" {
  group_object_id  = azuread_group.test.object_id
  member_object_id = azuread_service_principal.test.object_id
}
`, GroupResource{}.basic(data), ServicePrincipalResource{}.basic(data))
}

func (GroupMemberResource) oneUser(data acceptance.TestData) string {
	return fmt.Sprintf(`
%[1]s
%[2]s

resource "azuread_group_member" "testA" {
  group_object_id  = azuread_group.test.object_id
  member_object_id = azuread_user.testA.object_id
}
`, GroupResource{}.basic(data), UserResource{}.threeUsersABC(data))
}

func (GroupMemberResource) twoUsers(data acceptance.TestData) string {
	return fmt.Sprintf(`
%[1]s
%[2]s

resource "azuread_group_member" "testA" {
  group_object_id  = azuread_group.test.object_id
  member_object_id = azuread_user.testA.object_id
}

resource "azuread_group_member" "testB" {
  group_object_id  = azuread_group.test.object_id
  member_object_id = azuread_user.testB.object_id
}
`, GroupResource{}.basic(data), UserResource{}.threeUsersABC(data))
}

func (r GroupMemberResource) requiresImport(data acceptance.TestData) string {
	return fmt.Sprintf(`
%[1]s

resource "azuread_group_member" "import" {
  group_object_id  = azuread_group_member.test.group_object_id
  member_object_id = azuread_group_member.test.member_object_id
}
`, r.group(data))
}
