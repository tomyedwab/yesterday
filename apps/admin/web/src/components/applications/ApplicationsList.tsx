import { Box, Table, Text, Badge, HStack, Spinner, Alert } from "@chakra-ui/react";
import { LuServer } from "react-icons/lu";
import { useApplicationsView, type Application } from "../../dataviews/applications";

export const ApplicationsList = () => {
  const [loading, applications] = useApplicationsView();

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
        <HStack gap={2}>
          <LuServer />
          <Text fontSize="sm" color="gray.600">
            {applications.length} application{applications.length !== 1 ? 's' : ''}
          </Text>
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
            </Table.Row>
          ))}
        </Table.Body>
      </Table.Root>
    </Box>
  );
};