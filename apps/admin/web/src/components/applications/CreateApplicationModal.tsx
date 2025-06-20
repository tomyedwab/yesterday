import { useState, useEffect } from "react";
import {
  DialogRoot,
  DialogContent,
  DialogHeader,
  DialogBody,
  DialogFooter,
  DialogTitle,
  DialogCloseTrigger,
} from "@chakra-ui/react";
import { Button, Input, VStack, Text, Alert, HStack } from "@chakra-ui/react";
import { LuX } from "react-icons/lu";
import {
  useCreateApplication,
  type CreateApplicationRequest,
} from "../../dataviews/applicationActions";

interface CreateApplicationModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export const CreateApplicationModal = ({
  isOpen,
  onClose,
  onSuccess,
}: CreateApplicationModalProps) => {
  const [appId, setAppId] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [hostName, setHostName] = useState("");
  const [error, setError] = useState<string | null>(null);

  const { createApplication, isLoading } = useCreateApplication();

  // Reset form when modal opens/closes
  useEffect(() => {
    if (isOpen) {
      setAppId("");
      setDisplayName("");
      setHostName("");
      setError(null);
    }
  }, [isOpen]);

  const validateAppId = (id: string): string | null => {
    if (!id.trim()) {
      return "App ID is required";
    }
    return null;
  };

  const validateDisplayName = (name: string): string | null => {
    if (!name.trim()) {
      return "Display name is required";
    }
    if (name.length < 3) {
      return "Display name must be at least 3 characters";
    }
    return null;
  };

  const validateHostName = (host: string): string | null => {
    if (!host.trim()) {
      return "Host name is required";
    }
    // Basic hostname validation
    if (!/^[a-zA-Z0-9.-:]+$/.test(host)) {
      return "Host name can only contain letters, numbers, dots, and hyphens";
    }
    return null;
  };

  const handleSubmit = async () => {
    // Validate all fields
    const appIdError = validateAppId(appId);
    if (appIdError) {
      setError(appIdError);
      return;
    }

    const displayNameError = validateDisplayName(displayName);
    if (displayNameError) {
      setError(displayNameError);
      return;
    }

    const hostNameError = validateHostName(hostName);
    if (hostNameError) {
      setError(hostNameError);
      return;
    }

    const request: CreateApplicationRequest = {
      appId: appId.trim(),
      displayName: displayName.trim(),
      hostName: hostName.trim(),
    };

    const result = await createApplication(request);

    if (result.success) {
      onSuccess();
    } else {
      setError(result.error || "Failed to create application");
    }
  };

  const handleClose = () => {
    if (!isLoading) {
      onClose();
    }
  };

  const canSubmit =
    appId.trim() &&
    displayName.trim() &&
    hostName.trim() &&
    validateAppId(appId) === null &&
    validateDisplayName(displayName) === null &&
    validateHostName(hostName) === null;

  return (
    <DialogRoot open={isOpen} onOpenChange={(e) => !e.open && handleClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create New Application</DialogTitle>
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
              <Text fontWeight="medium">App ID</Text>
              <Input
                value={appId}
                onChange={(e) => setAppId(e.target.value)}
                placeholder="e.g., 0001-0003"
                disabled={isLoading}
                autoFocus
                fontFamily="mono"
              />
            </VStack>

            <VStack gap={2} align="stretch">
              <Text fontWeight="medium">Display Name</Text>
              <Input
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                placeholder="e.g., My Application"
                disabled={isLoading}
              />
              <Text fontSize="xs" color="gray.500">
                Human-readable name for the application
              </Text>
            </VStack>

            <VStack gap={2} align="stretch">
              <Text fontWeight="medium">Host Name</Text>
              <Input
                value={hostName}
                onChange={(e) => setHostName(e.target.value)}
                placeholder="e.g., myapp.example.com"
                disabled={isLoading}
                fontFamily="mono"
              />
              <Text fontSize="xs" color="gray.500">
                Hostname where the application is accessible
              </Text>
            </VStack>

            <Alert.Root status="info">
              <Alert.Indicator />
              <Alert.Content>
                <Alert.Title>Application Registration</Alert.Title>
                <Alert.Description>
                  This will register a new application in the system.
                </Alert.Description>
              </Alert.Content>
            </Alert.Root>
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
              Create Application
            </Button>
          </HStack>
        </DialogFooter>
      </DialogContent>
    </DialogRoot>
  );
};
