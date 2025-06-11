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
import { useCreateUser, type CreateUserRequest } from '../../dataviews/userActions';

interface CreateUserModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export const CreateUserModal = ({ isOpen, onClose, onSuccess }: CreateUserModalProps) => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  const { createUser, isLoading } = useCreateUser();

  // Reset form when modal opens/closes
  useEffect(() => {
    if (isOpen) {
      setUsername('');
      setPassword('');
      setConfirmPassword('');
      setShowPassword(false);
      setShowConfirmPassword(false);
      setError(null);
    }
  }, [isOpen]);

  const validateUsername = (name: string): string | null => {
    if (!name.trim()) {
      return 'Username is required';
    }
    if (name.length < 3) {
      return 'Username must be at least 3 characters';
    }
    if (!/^[a-zA-Z0-9_-]+$/.test(name)) {
      return 'Username can only contain letters, numbers, hyphens, and underscores';
    }
    return null;
  };

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
    // Validate username
    const usernameError = validateUsername(username);
    if (usernameError) {
      setError(usernameError);
      return;
    }

    // Validate password
    const passwordError = validatePassword(password);
    if (passwordError) {
      setError(passwordError);
      return;
    }

    // Check password confirmation
    if (password !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    const request: CreateUserRequest = {
      username: username.trim(),
      password,
    };

    const result = await createUser(request);
    
    if (result.success) {
      onSuccess();
    } else {
      setError(result.error || 'Failed to create user');
    }
  };

  const handleClose = () => {
    if (!isLoading) {
      onClose();
    }
  };

  const canSubmit = username.trim() && password && confirmPassword && password === confirmPassword && validatePassword(password) === null;

  return (
    <DialogRoot open={isOpen} onOpenChange={(e) => !e.open && handleClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create New User</DialogTitle>
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
              <Text fontWeight="medium">Username</Text>
              <Input
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="Enter username"
                disabled={isLoading}
                autoFocus
              />
            </VStack>

            <VStack gap={2} align="stretch">
              <Text fontWeight="medium">Password</Text>
              <HStack>
                <Input
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Enter password"
                  disabled={isLoading}
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
                  placeholder="Confirm password"
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

            <Text fontSize="sm" color="gray.600">
              The user will be created with the specified username and password.
              Password must be at least 8 characters with uppercase, lowercase, and a number.
            </Text>
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
              disabled={!canSubmit}
            >
              Create User
            </Button>
          </HStack>
        </DialogFooter>
      </DialogContent>
    </DialogRoot>
  );
};