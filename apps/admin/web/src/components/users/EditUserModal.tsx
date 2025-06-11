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
  Input,
  VStack,
  Text,
  Alert,
  HStack,
} from '@chakra-ui/react';
import { LuX } from 'react-icons/lu';
import { useUpdateUser, type UpdateUserRequest } from '../../dataviews/userActions';
import type { User } from '../../dataviews/users';

interface EditUserModalProps {
  isOpen: boolean;
  onClose: () => void;
  user: User | null;
  onSuccess?: () => void;
}

export const EditUserModal = ({ isOpen, onClose, user, onSuccess }: EditUserModalProps) => {
  const [username, setUsername] = useState('');
  const [error, setError] = useState<string | null>(null);
  const { updateUser, isLoading } = useUpdateUser();

  useEffect(() => {
    if (user) {
      setUsername(user.username);
    } else {
      setUsername('');
    }
    setError(null);
  }, [user, isOpen]);

  const handleSubmit = async () => {
    if (!user) return;
    
    if (!username.trim()) {
      setError('Username is required');
      return;
    }

    if (username === user.username) {
      onClose();
      return;
    }

    const request: UpdateUserRequest = {
      userID: user.id,
      username: username.trim(),
    };

    const result = await updateUser(request);
    
    if (result.success) {
      onClose();
      onSuccess?.();
    } else {
      setError(result.error || 'Failed to update user');
    }
  };

  const handleClose = () => {
    if (!isLoading) {
      onClose();
    }
  };

  return (
    <DialogRoot open={isOpen} onOpenChange={(e) => !e.open && handleClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit User</DialogTitle>
          <DialogCloseTrigger asChild>
            <Button variant="ghost" size="sm" disabled={isLoading}>
              <LuX />
            </Button>
          </DialogCloseTrigger>
        </DialogHeader>

        <DialogBody>
          <VStack gap={4} align="stretch">
            {error && (
              <Alert.Root status="error">
                <Alert.Indicator />
                <Alert.Content>
                  <Alert.Title>Error</Alert.Title>
                  <Alert.Description>{error}</Alert.Description>
                </Alert.Content>
              </Alert.Root>
            )}

            {user?.id === 1 && (
              <Alert.Root status="info">
                <Alert.Indicator />
                <Alert.Content>
                  <Alert.Title>Admin User Protection</Alert.Title>
                  <Alert.Description>
                    The admin user's username cannot be changed for security reasons.
                  </Alert.Description>
                </Alert.Content>
              </Alert.Root>
            )}

            <VStack gap={2} align="stretch">
              <Text fontWeight="medium">User ID</Text>
              <Text fontFamily="mono" color="gray.600" fontSize="sm">
                {user?.id}
              </Text>
            </VStack>

            <VStack gap={2} align="stretch">
              <Text fontWeight="medium">Username</Text>
              <Input
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="Enter username"
                disabled={isLoading || user?.id === 1}
                autoFocus
              />
            </VStack>
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
              colorScheme="blue"
              onClick={handleSubmit}
              loading={isLoading}
              disabled={!username.trim() || username === user?.username || user?.id === 1}
            >
              Save Changes
            </Button>
          </HStack>
        </DialogFooter>
      </DialogContent>
    </DialogRoot>
  );
};