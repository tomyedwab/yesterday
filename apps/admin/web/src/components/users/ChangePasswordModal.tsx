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
import { LuX, LuEye, LuEyeOff, LuCheck } from 'react-icons/lu';
import { useUpdatePassword, type UpdatePasswordRequest } from '../../dataviews/userActions';
import type { User } from '../../dataviews/users';

interface ChangePasswordModalProps {
  isOpen: boolean;
  onClose: () => void;
  user: User | null;
  onSuccess?: () => void;
}

export const ChangePasswordModal = ({ isOpen, onClose, user, onSuccess }: ChangePasswordModalProps) => {
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { updatePassword, isLoading } = useUpdatePassword();

  useEffect(() => {
    if (isOpen) {
      setPassword('');
      setConfirmPassword('');
      setShowPassword(false);
      setShowConfirmPassword(false);
      setError(null);
    }
  }, [isOpen]);

  const validatePassword = (pwd: string): string | null => {
    if (pwd.length < 8) {
      return 'Password must be at least 8 characters long';
    }
    if (!/[A-Z]/.test(pwd)) {
      return 'Password must contain at least one uppercase letter';
    }
    if (!/[a-z]/.test(pwd)) {
      return 'Password must contain at least one lowercase letter';
    }
    if (!/[0-9]/.test(pwd)) {
      return 'Password must contain at least one number';
    }
    return null;
  };

  const getPasswordValidation = (pwd: string) => {
    if (!pwd) return { isValid: false, showValidation: false };
    
    const validationError = validatePassword(pwd);
    return {
      isValid: validationError === null,
      showValidation: true,
      message: validationError || 'Password meets all requirements'
    };
  };

  const passwordValidation = getPasswordValidation(password);

  const handleSubmit = async () => {
    if (!user) return;
    
    if (!password) {
      setError('Password is required');
      return;
    }

    const validationError = validatePassword(password);
    if (validationError) {
      setError(validationError);
      return;
    }

    if (password !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    const request: UpdatePasswordRequest = {
      userID: user.id,
      newPassword: password,
    };

    const result = await updatePassword(request);
    
    if (result.success) {
      onClose();
      onSuccess?.();
    } else {
      setError(result.error || 'Failed to update password');
    }
  };

  const handleClose = () => {
    if (!isLoading) {
      onClose();
    }
  };

  const canSubmit = password && confirmPassword && password === confirmPassword && validatePassword(password) === null;

  return (
    <DialogRoot open={isOpen} onOpenChange={(e) => !e.open && handleClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Change Password</DialogTitle>
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

            <VStack gap={2} align="stretch">
              <Text fontWeight="medium">User</Text>
              <Text color="gray.600" fontSize="sm">
                {user?.username} (ID: {user?.id})
              </Text>
            </VStack>

            <VStack gap={2} align="stretch">
              <Text fontWeight="medium">New Password</Text>
              <HStack>
                <Input
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Enter new password"
                  disabled={isLoading}
                  autoFocus
                />
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setShowPassword(!showPassword)}
                  disabled={isLoading}
                >
                  {showPassword ? <LuEyeOff /> : <LuEye />}
                </Button>
              </HStack>
              {passwordValidation.showValidation && (
                <HStack gap={2} align="center">
                  {passwordValidation.isValid ? (
                    <LuCheck color="green" size={16} />
                  ) : (
                    <LuX color="red" size={16} />
                  )}
                  <Text 
                    fontSize="xs" 
                    color={passwordValidation.isValid ? "green.600" : "red.500"}
                  >
                    {passwordValidation.message}
                  </Text>
                </HStack>
              )}
              {!passwordValidation.showValidation && (
                <Text fontSize="xs" color="gray.500">
                  Password must be at least 8 characters with uppercase, lowercase, and number
                </Text>
              )}
            </VStack>

            <VStack gap={2} align="stretch">
              <Text fontWeight="medium">Confirm Password</Text>
              <HStack>
                <Input
                  type={showConfirmPassword ? 'text' : 'password'}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder="Confirm new password"
                  disabled={isLoading}
                />
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  disabled={isLoading}
                >
                  {showConfirmPassword ? <LuEyeOff /> : <LuEye />}
                </Button>
              </HStack>
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
              colorScheme="orange"
              onClick={handleSubmit}
              loading={isLoading}
              disabled={!canSubmit}
            >
              Update Password
            </Button>
          </HStack>
        </DialogFooter>
      </DialogContent>
    </DialogRoot>
  );
};