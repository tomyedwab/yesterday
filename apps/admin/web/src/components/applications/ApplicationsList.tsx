import { Box, Table, Text, Badge, HStack, Spinner, Alert, Button, IconButton } from "@chakra-ui/react";
import { toaster } from "../ui/toaster";
import { LuServer, LuPlus, LuPencil, LuTrash2 } from "react-icons/lu";
import { useState } from "react";
import { useApplicationsView, type Application } from "../../dataviews/applications";
import { CreateApplicationModal } from "./CreateApplicationModal";
import { EditApplicationModal } from "./EditApplicationModal";
import { DeleteApplicationModal } from "./DeleteApplicationModal";

export const ApplicationsList = () => {
  const [loading, applications] = useApplicationsView();
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [selectedApplication, setSelectedApplication] = useState<Application | null>(null);

  const closeModals = () => {
    setCreateModalOpen(false);
    setEditModalOpen(false);
    setDeleteModalOpen(false);
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

  if (applications.length === 0) {
    return (
      <Box>
        <HStack justify="space-between" mb={4}>
          <Text fontSize="lg" fontWeight="medium">Applications</Text>
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

  const getApplicationTypeBadge = (appId: string, displayName: string) => {
    if (appId === "0001-0001" || displayName.toLowerCase().includes("login")) {
      return (
        <Badge colorScheme="blue" variant="subtle">
          Login Service
        </Badge>
      );
    }
    if (appId === "0001-0002" || displayName.toLowerCase().includes("admin")) {
      return (
        <Badge colorScheme="purple" variant="subtle">
          Admin Service
        </Badge>
      );
    }
    return (
      <Badge colorScheme="green" variant="subtle">
        Application
      </Badge>
    );
  };

  return (
    <Box>
      <HStack justify="space-between" mb={4}>
        <Text fontSize="lg" fontWeight="medium">Applications</Text>
        <HStack gap={4}>
          <HStack gap={2}>
            <LuServer />
            <Text fontSize="sm" color="gray.600">
              {applications.length} application{applications.length !== 1 ? 's' : ''}
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
            <Table.ColumnHeader>Database</Table.ColumnHeader>
            <Table.ColumnHeader>Type</Table.ColumnHeader>
            <Table.ColumnHeader>Instance ID</Table.ColumnHeader>
            <Table.ColumnHeader width="120px">Actions</Table.ColumnHeader>
          </Table.Row>
        </Table.Header>
        <Table.Body>
          {applications.map((app) => (
            <Table.Row key={app.instanceId}>
              <Table.Cell>
                <Text fontWeight="medium">{app.displayName}</Text>
              </Table.Cell>
              <Table.Cell>
                <Text fontFamily="mono" fontSize="sm">{app.appId}</Text>
              </Table.Cell>
              <Table.Cell>
                <Text fontSize="sm">{app.hostName}</Text>
              </Table.Cell>
              <Table.Cell>
                <Text fontFamily="mono" fontSize="sm">{app.dbName}</Text>
              </Table.Cell>
              <Table.Cell>
                {getApplicationTypeBadge(app.appId, app.displayName)}
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