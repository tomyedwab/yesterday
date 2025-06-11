import { useState, useEffect } from 'react';
import {
  DialogRoot,
  DialogContent,
  DialogHeader,
  DialogBody,
  DialogFooter,
  DialogTitle,
  DialogCloseTrigger,
} from '@chakra-ui/react';
import {
  Button,
  VStack,
  Text,
  Alert,
  HStack,
  Badge,
} from '@chakra-ui/react';
import { LuX } from 'react-icons/lu';
import { useDeleteUser, type DeleteUserRequest } from '../../dataviews/userActions';
import type { User } from '../../dataviews/users';

interface DeleteUserModalProps {
  isOpen: boolean;
  onClose: () => void;
  user: User | null;
  onSuccess?: () => void;
}

export const DeleteUserModal = ({ isOpen, onClose, user, onSuccess }: DeleteUserModalProps) => {
  const [error, setError] = useState<string | null>(null);
  const { deleteUser, isLoading } = useDeleteUser();

  useEffect(() => {
    if (isOpen) {
      setError(null);
    }
  }, [isOpen]);

  const handleSubmit = async () => {
    if (!user) return;
    
    const request: DeleteUserRequest = {
      userID: user.id,
    };

    const result = await deleteUser(request);
    
    if (result.success) {
      onClose();
      onSuccess?.();
    } else {
      setError(result.error || 'Failed to delete user');
    }
  };

  const handleClose = () => {
    if (!isLoading) {
      onClose();
    }
  };

  const isAdminUser = user?.username === 'admin';

  return (
    <DialogRoot open={isOpen} onOpenChange={(e) => !e.open && handleClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete User</DialogTitle>
          <DialogCloseTrigger asChild>
            <Button variant="ghost" size="sm" disabled={isLoading}>
              <LuX />
            </Button>
          </DialogCloseTrigger>
        </DialogHeader>

        <DialogBody>
          <VStack gap={4} align="stretch">
            <Alert.Root status="warning">
              <Alert.Indicator />
              <Alert.Content>
                <Alert.Title>Confirm Deletion</Alert.Title>
                <Alert.Description>
                  This action cannot be undone. All user access rules will also be deleted.
                </Alert.Description>
              </Alert.Content>
            </Alert.Root>

            {error && (
              <Alert.Root status="error">
                <Alert.Indicator />
                <Alert.Content>
                  <Alert.Title>Error</Alert.Title>
                  <Alert.Description>{error}</Alert.Description>
                </Alert.Content>
              </Alert.Root>
            )}

            {isAdminUser && (
              <Alert.Root status="error">
                <Alert.Indicator />
                <Alert.Content>
                  <Alert.Title>Cannot Delete Admin User</Alert.Title>
                  <Alert.Description>
                    The admin user cannot be deleted for security reasons.
                  </Alert.Description>
                </Alert.Content>
              </Alert.Root>
            )}

            <VStack gap={3} align="stretch" p={4} bg="gray.50" borderRadius="md">
              <HStack justify="space-between">
                <Text fontWeight="medium" color="gray.700">User ID:</Text>
                <Text fontFamily="mono" color="gray.900">{user?.id}</Text>
              </HStack>
              <HStack justify="space-between">
                <Text fontWeight="medium" color="gray.700">Username:</Text>
                <HStack>
                  <Text color="gray.900">{user?.username}</Text>
                  {isAdminUser && (
                    <Badge colorScheme="blue" variant="subtle">
                      Admin
                    </Badge>
                  )}
                </HStack>
              </HStack>
            </VStack>

            {!isAdminUser && (
              <Text fontSize="sm" color="gray.600">
                Are you sure you want to delete this user? This will permanently remove:
              </Text>
            )}

            {!isAdminUser && (
              <VStack gap={1} align="stretch" fontSize="sm" color="gray.600" pl={4}>
                <Text>• The user account</Text>
                <Text>• All associated access rules</Text>
                <Text>• Any user-specific data</Text>
              </VStack>
            )}
          </VStack>
        </DialogBody>

        <DialogFooter>
          <HStack gap={3}>
            <Button
              variant="outline"
              onClick={handleClose}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button
              colorScheme="red"
              onClick={handleSubmit}
              loading={isLoading}
              disabled={isAdminUser}
            >
              Delete User
            </Button>
          </HStack>
        </DialogFooter>
      </DialogContent>
    </DialogRoot>
  );
};