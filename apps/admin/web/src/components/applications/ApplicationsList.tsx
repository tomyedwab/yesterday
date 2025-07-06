import {
  Box,
  Table,
  Text,
  Badge,
  HStack,
  VStack,
  Spinner,
  Alert,
  Button,
  IconButton,
} from "@chakra-ui/react";
import { toaster } from "../ui/toaster";
import { LuServer, LuPlus, LuPencil, LuTrash2, LuShield, LuBug, LuKey } from "react-icons/lu";
import { useState } from "react";
import {
  useApplicationsView,
  type Application,
} from "../../dataviews/applications";
import { CreateApplicationModal } from "./CreateApplicationModal";
import { EditApplicationModal } from "./EditApplicationModal";
import { DeleteApplicationModal } from "./DeleteApplicationModal";
import { UserAccessRulesList } from "../access-rules/UserAccessRulesList";

export const ApplicationsList = () => {
  const [loading, applications] = useApplicationsView();
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [accessRulesView, setAccessRulesView] = useState(false);
  const [selectedApplication, setSelectedApplication] =
    useState<Application | null>(null);

  const closeModals = () => {
    setCreateModalOpen(false);
    setEditModalOpen(false);
    setDeleteModalOpen(false);
    setAccessRulesView(false);
    setSelectedApplication(null);
  };

  const handleSuccess = (action: string) => {
    closeModals();

    // Show success toast
    toaster.create({
      title: "Success",
      description: `Application ${action} successfully`,
      duration: 3000,
    });

    // The Yesterday framework handles data refresh automatically via events
  };

  const handleCreateSuccess = () => {
    handleSuccess("created");
  };

  const handleEditClick = (application: Application) => {
    setSelectedApplication(application);
    setEditModalOpen(true);
  };

  const handleEditSuccess = () => {
    handleSuccess("updated");
  };

  const handleDeleteClick = (application: Application) => {
    setSelectedApplication(application);
    setDeleteModalOpen(true);
  };

  const handleDeleteSuccess = () => {
    handleSuccess("deleted");
  };

  const handleAccessRulesClick = (application: Application) => {
    setSelectedApplication(application);
    setAccessRulesView(true);
  };

  const handleBackToApplications = () => {
    setAccessRulesView(false);
    setSelectedApplication(null);
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" p={8}>
        <HStack gap={2}>
          <Spinner size="sm" />
          <Text>Loading applications...</Text>
        </HStack>
      </Box>
    );
  }

  // Show access rules view if selected
  if (accessRulesView && selectedApplication) {
    return (
      <Box>
        <HStack justify="space-between" mb={4}>
          <HStack gap={2}>
            <Button
              variant="ghost"
              size="sm"
              onClick={handleBackToApplications}
            >
              ← Back to Applications
            </Button>
            <Text fontSize="lg" fontWeight="medium">
              {selectedApplication.displayName} - Access Rules
            </Text>
          </HStack>
        </HStack>
        <UserAccessRulesList
          applicationId={selectedApplication.instanceId}
          applicationName={selectedApplication.displayName}
        />
      </Box>
    );
  }

  if (applications.length === 0) {
    return (
      <Box>
        <HStack justify="space-between" mb={4}>
          <Text fontSize="lg" fontWeight="medium">
            Applications
          </Text>
          <Button
            colorScheme="blue"
            size="sm"
            onClick={() => setCreateModalOpen(true)}
          >
            <LuPlus />
            Create Application
          </Button>
        </HStack>
        <Alert.Root status="info">
          <Alert.Indicator />
          <Alert.Content>
            <Alert.Title>No applications found</Alert.Title>
            <Alert.Description>
              No applications are currently registered in the system.
            </Alert.Description>
          </Alert.Content>
        </Alert.Root>
      </Box>
    );
  }



  return (
    <Box>
      <HStack justify="space-between" mb={4}>
        <Text fontSize="lg" fontWeight="medium">
          Applications
        </Text>
        <HStack gap={4}>
          <HStack gap={2}>
            <LuServer />
            <Text fontSize="sm" color="gray.600">
              {applications.length} application
              {applications.length !== 1 ? "s" : ""}
            </Text>
          </HStack>
          <Button
            colorScheme="blue"
            size="sm"
            onClick={() => setCreateModalOpen(true)}
          >
            <LuPlus />
            Create Application
          </Button>
        </HStack>
      </HStack>
      <Table.Root size="md" variant="outline">
        <Table.Header>
          <Table.Row>
            <Table.ColumnHeader>Display Name</Table.ColumnHeader>
            <Table.ColumnHeader>App ID</Table.ColumnHeader>
            <Table.ColumnHeader>Host Name</Table.ColumnHeader>
            <Table.ColumnHeader>Debug Info</Table.ColumnHeader>
            <Table.ColumnHeader>Instance ID</Table.ColumnHeader>
            <Table.ColumnHeader width="160px">Actions</Table.ColumnHeader>
          </Table.Row>
        </Table.Header>
        <Table.Body>
          {applications.map((app) => (
            <Table.Row key={app.instanceId}>
              <Table.Cell>
                <HStack>
                  <Text fontWeight="medium">{app.displayName}</Text>
                  {app.debugPublishToken && (
                    <Badge colorPalette="orange" size="sm" variant="subtle">
                      <LuBug size={12} />
                      Debug
                    </Badge>
                  )}
                </HStack>
              </Table.Cell>
              <Table.Cell>
                <Text fontFamily="mono" fontSize="sm" color="gray.600">
                  {app.appId}
                </Text>
              </Table.Cell>
              <Table.Cell>
                <Text fontFamily="mono" fontSize="sm" color="gray.600">
                  {app.hostName}
                </Text>
              </Table.Cell>
              <Table.Cell>
                <VStack align="start" gap={1}>
                  {app.debugPublishToken && (
                    <HStack gap={1}>
                      <LuKey size={12} color="gray.500" />
                      <Text fontFamily="mono" fontSize="xs" color="gray.500">
                        Token: {app.debugPublishToken}
                      </Text>
                    </HStack>
                  )}
                  {app.debugPublishToken && app.staticServiceUrl && (
                    <Text fontFamily="mono" fontSize="xs" color="blue.500" title="Static Service URL">
                      → {app.staticServiceUrl}
                    </Text>
                  )}
                  {!app.debugPublishToken && (
                    <Text fontSize="xs" color="gray.400">
                      Production
                    </Text>
                  )}
                </VStack>
              </Table.Cell>
              <Table.Cell>
                <Text fontFamily="mono" fontSize="xs" color="gray.500">
                  {app.instanceId}
                </Text>
              </Table.Cell>
              <Table.Cell>
                <HStack gap={1}>
                  <IconButton
                    size="sm"
                    variant="ghost"
                    aria-label="Manage access rules"
                    onClick={() => handleAccessRulesClick(app)}
                  >
                    <LuShield />
                  </IconButton>
                  <IconButton
                    size="sm"
                    variant="ghost"
                    aria-label="Edit application"
                    onClick={() => handleEditClick(app)}
                  >
                    <LuPencil />
                  </IconButton>
                  <IconButton
                    size="sm"
                    variant="ghost"
                    colorScheme="red"
                    aria-label="Delete application"
                    onClick={() => handleDeleteClick(app)}
                  >
                    <LuTrash2 />
                  </IconButton>
                </HStack>
              </Table.Cell>
            </Table.Row>
          ))}
        </Table.Body>
      </Table.Root>

      <CreateApplicationModal
        isOpen={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        onSuccess={handleCreateSuccess}
      />

      <EditApplicationModal
        isOpen={editModalOpen}
        onClose={() => setEditModalOpen(false)}
        application={selectedApplication}
        onSuccess={handleEditSuccess}
      />

      <DeleteApplicationModal
        isOpen={deleteModalOpen}
        onClose={() => setDeleteModalOpen(false)}
        application={selectedApplication}
        onSuccess={handleDeleteSuccess}
      />
    </Box>
  );
};
