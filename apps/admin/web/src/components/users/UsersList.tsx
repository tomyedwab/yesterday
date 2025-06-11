import { Box, Table, Text, Badge, Button, HStack, Spinner, Alert } from "@chakra-ui/react";
import { LuPencilLine, LuTrash2, LuKey } from "react-icons/lu";
import { useUsersView } from "../../dataviews/users";

export const UsersList = () => {
  const [loading, users] = useUsersView();

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" p={8}>
        <HStack gap={2}>
          <Spinner size="sm" />
          <Text>Loading users...</Text>
        </HStack>
      </Box>
    );
  }

  if (users.length === 0) {
    return (
      <Alert.Root status="info">
        <Alert.Indicator />
        <Alert.Content>
          <Alert.Title>No users found</Alert.Title>
          <Alert.Description>
            Create your first user to get started.
          </Alert.Description>
        </Alert.Content>
      </Alert.Root>
    );
  }

  return (
    <Box>
      <Table.Root size="md" variant="outline">
        <Table.Header>
          <Table.Row>
            <Table.ColumnHeader>ID</Table.ColumnHeader>
            <Table.ColumnHeader>Username</Table.ColumnHeader>
            <Table.ColumnHeader>Status</Table.ColumnHeader>
            <Table.ColumnHeader textAlign="end">Actions</Table.ColumnHeader>
          </Table.Row>
        </Table.Header>
        <Table.Body>
          {users.map((user) => (
            <Table.Row key={user.id}>
              <Table.Cell>
                <Text fontFamily="mono">{user.id}</Text>
              </Table.Cell>
              <Table.Cell>
                <Text fontWeight="medium">{user.username}</Text>
              </Table.Cell>
              <Table.Cell>
                <Badge
                  colorScheme={user.username === "admin" ? "blue" : "green"}
                  variant="subtle"
                >
                  {user.username === "admin" ? "Admin" : "User"}
                </Badge>
              </Table.Cell>
              <Table.Cell textAlign="end">
                <HStack justify="flex-end" gap={2}>
                  <Button
                    size="sm"
                    variant="ghost"
                    colorScheme="blue"
                    disabled
                  >
                    <LuPencilLine />
                    Edit
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    colorScheme="orange"
                    disabled
                  >
                    <LuKey />
                    Password
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    colorScheme="red"
                    disabled={user.username === "admin"}
                  >
                    <LuTrash2 />
                    Delete
                  </Button>
                </HStack>
              </Table.Cell>
            </Table.Row>
          ))}
        </Table.Body>
      </Table.Root>
    </Box>
  );
};