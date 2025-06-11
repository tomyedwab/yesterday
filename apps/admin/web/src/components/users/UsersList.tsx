import { Box, Table, Text, Badge, Button, HStack, Spinner, Alert } from "@chakra-ui/react";
import { LuPencilLine, LuTrash2, LuKey } from "react-icons/lu";
import { useState } from "react";
import { useUsersView, type User } from "../../dataviews/users";
import { EditUserModal } from "./EditUserModal";
import { ChangePasswordModal } from "./ChangePasswordModal";
import { DeleteUserModal } from "./DeleteUserModal";
import { toaster } from "../ui/toaster";

export const UsersList = () => {
  const [loading, users] = useUsersView();
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);

  const handleEditUser = (user: User) => {
    setSelectedUser(user);
    setEditModalOpen(true);
  };

  const handleChangePassword = (user: User) => {
    setSelectedUser(user);
    setPasswordModalOpen(true);
  };

  const handleDeleteUser = (user: User) => {
    setSelectedUser(user);
    setDeleteModalOpen(true);
  };

  const closeModals = () => {
    setEditModalOpen(false);
    setPasswordModalOpen(false);
    setDeleteModalOpen(false);
    setSelectedUser(null);
  };

  const handleSuccess = (action: string) => {
    closeModals();
    
    // Show success toast
    toaster.create({
      title: "Success",
      description: `User ${action} successfully`,
      duration: 3000,
    });
    
    // The Yesterday framework handles data refresh automatically via events
  };



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
                    onClick={() => handleEditUser(user)}
                    disabled={user.id === 1}
                  >
                    <LuPencilLine />
                    Edit
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    colorScheme="orange"
                    onClick={() => handleChangePassword(user)}
                  >
                    <LuKey />
                    Password
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    colorScheme="red"
                    disabled={user.username === "admin"}
                    onClick={() => handleDeleteUser(user)}
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
      {/* Modals */}
      <EditUserModal
        isOpen={editModalOpen}
        onClose={closeModals}
        user={selectedUser}
        onSuccess={() => handleSuccess("updated")}
      />
      
      <ChangePasswordModal
        isOpen={passwordModalOpen}
        onClose={closeModals}
        user={selectedUser}
        onSuccess={() => handleSuccess("password changed")}
      />
      
      <DeleteUserModal
        isOpen={deleteModalOpen}
        onClose={closeModals}
        user={selectedUser}
        onSuccess={() => handleSuccess("deleted")}
      />
    </Box>
  );
};