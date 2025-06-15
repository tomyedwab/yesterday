import {
  Box,
  Table,
  Text,
  Badge,
  HStack,
  Spinner,
  Alert,
  Button,
  IconButton,
} from "@chakra-ui/react";
import { toaster } from "../ui/toaster";
import {
  LuShield,
  LuPlus,
  LuPencil,
  LuTrash2,
  LuUser,
  LuUsers,
} from "react-icons/lu";
import { useState } from "react";
import {
  useUserAccessRulesView,
  type UserAccessRule,
} from "../../dataviews/userAccessRules";
import { useUsersView, type User } from "../../dataviews/users";
import { CreateUserAccessRuleModal } from "./CreateUserAccessRuleModal";
import { EditUserAccessRuleModal } from "./EditUserAccessRuleModal";
import { DeleteUserAccessRuleModal } from "./DeleteUserAccessRuleModal";

interface UserAccessRulesListProps {
  applicationId: string;
  applicationName: string;
}

export const UserAccessRulesList = ({
  applicationId,
  applicationName,
}: UserAccessRulesListProps) => {
  const [loading, rules] = useUserAccessRulesView(applicationId);
  const [loadingUsers, users] = useUsersView();
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [selectedRule, setSelectedRule] = useState<UserAccessRule | null>(null);

  const closeModals = () => {
    setCreateModalOpen(false);
    setEditModalOpen(false);
    setDeleteModalOpen(false);
    setSelectedRule(null);
  };

  const handleSuccess = (action: string) => {
    closeModals();

    // Show success toast
    toaster.create({
      title: "Success",
      description: `Access rule ${action} successfully`,
      duration: 3000,
    });

    // The Yesterday framework handles data refresh automatically via events
  };

  const handleCreateSuccess = () => {
    handleSuccess("created");
  };

  const handleEditClick = (rule: UserAccessRule) => {
    setSelectedRule(rule);
    setEditModalOpen(true);
  };

  const handleEditSuccess = () => {
    handleSuccess("updated");
  };

  const handleDeleteClick = (rule: UserAccessRule) => {
    setSelectedRule(rule);
    setDeleteModalOpen(true);
  };

  const handleDeleteSuccess = () => {
    handleSuccess("deleted");
  };

  const getRuleTypeBadge = (ruleType: string) => {
    if (ruleType === "ACCEPT") {
      return (
        <Badge colorScheme="green" variant="subtle">
          Allow
        </Badge>
      );
    }
    return (
      <Badge colorScheme="red" variant="subtle">
        Deny
      </Badge>
    );
  };

  const getSubjectIcon = (subjectType: string) => {
    return subjectType === "USER" ? <LuUser /> : <LuUsers />;
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return (
      date.toLocaleDateString() +
      " " +
      date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
    );
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" p={8}>
        <HStack gap={2}>
          <Spinner size="sm" />
          <Text>Loading access rules...</Text>
        </HStack>
      </Box>
    );
  }

  if (rules.length === 0) {
    return (
      <Box>
        <HStack justify="space-between" mb={4}>
          <Text fontSize="lg" fontWeight="medium">
            Access Rules
          </Text>
          <Button
            colorScheme="blue"
            size="sm"
            onClick={() => setCreateModalOpen(true)}
          >
            <LuPlus />
            Add Access Rule
          </Button>
        </HStack>
        <Alert.Root status="info">
          <Alert.Indicator />
          <Alert.Content>
            <Alert.Title>No access rules configured</Alert.Title>
            <Alert.Description>
              No access rules are currently configured for {applicationName}.
              Click "Add Access Rule" to create the first rule.
            </Alert.Description>
          </Alert.Content>
        </Alert.Root>
      </Box>
    );
  }

  const usersMap: Record<string, string> = {};
  if (!loadingUsers) {
    users.forEach((user: User) => {
      usersMap[user.id] = user.username;
    });
  }

  const getSubjectID = (row: UserAccessRule) => {
    if (row.subjectType === "USER") {
      if (loadingUsers) {
        return "Loading...";
      }
      return usersMap[row.subjectId];
    }
    return row.subjectType;
  };

  return (
    <Box>
      <HStack justify="space-between" mb={4}>
        <Text fontSize="lg" fontWeight="medium">
          Access Rules
        </Text>
        <HStack gap={4}>
          <HStack gap={2}>
            <LuShield />
            <Text fontSize="sm" color="gray.600">
              {rules.length} rule{rules.length !== 1 ? "s" : ""}
            </Text>
          </HStack>
          <Button
            colorScheme="blue"
            size="sm"
            onClick={() => setCreateModalOpen(true)}
          >
            <LuPlus />
            Add Access Rule
          </Button>
        </HStack>
      </HStack>

      <Table.Root size="md" variant="outline">
        <Table.Header>
          <Table.Row>
            <Table.ColumnHeader>Rule Type</Table.ColumnHeader>
            <Table.ColumnHeader>Subject Type</Table.ColumnHeader>
            <Table.ColumnHeader>Subject ID</Table.ColumnHeader>
            <Table.ColumnHeader>Created</Table.ColumnHeader>
            <Table.ColumnHeader width="120px">Actions</Table.ColumnHeader>
          </Table.Row>
        </Table.Header>
        <Table.Body>
          {rules.map((rule) => (
            <Table.Row key={rule.ruleId}>
              <Table.Cell>{getRuleTypeBadge(rule.ruleType)}</Table.Cell>
              <Table.Cell>
                <HStack gap={2}>
                  {getSubjectIcon(rule.subjectType)}
                  <Text fontSize="sm" textTransform="capitalize">
                    {rule.subjectType.toLowerCase()}
                  </Text>
                </HStack>
              </Table.Cell>
              <Table.Cell>
                <Text fontFamily="mono" fontSize="sm">
                  {getSubjectID(rule)}
                </Text>
              </Table.Cell>
              <Table.Cell>
                <Text fontSize="sm" color="gray.600">
                  {formatDate(rule.createdAt)}
                </Text>
              </Table.Cell>
              <Table.Cell>
                <HStack gap={1}>
                  <IconButton
                    size="sm"
                    variant="ghost"
                    aria-label="Edit access rule"
                    onClick={() => handleEditClick(rule)}
                  >
                    <LuPencil />
                  </IconButton>
                  <IconButton
                    size="sm"
                    variant="ghost"
                    colorScheme="red"
                    aria-label="Delete access rule"
                    onClick={() => handleDeleteClick(rule)}
                  >
                    <LuTrash2 />
                  </IconButton>
                </HStack>
              </Table.Cell>
            </Table.Row>
          ))}
        </Table.Body>
      </Table.Root>

      <CreateUserAccessRuleModal
        isOpen={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        applicationId={applicationId}
        applicationName={applicationName}
        onSuccess={handleCreateSuccess}
      />

      <EditUserAccessRuleModal
        isOpen={editModalOpen}
        onClose={() => setEditModalOpen(false)}
        rule={selectedRule}
        applicationName={applicationName}
        onSuccess={handleEditSuccess}
      />

      <DeleteUserAccessRuleModal
        isOpen={deleteModalOpen}
        onClose={() => setDeleteModalOpen(false)}
        rule={selectedRule}
        applicationName={applicationName}
        onSuccess={handleDeleteSuccess}
      />
    </Box>
  );
};
