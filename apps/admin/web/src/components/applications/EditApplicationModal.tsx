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
  useUpdateApplication,
  type UpdateApplicationRequest,
} from "../../dataviews/applicationActions";
import type { Application } from "../../dataviews/applications";

interface EditApplicationModalProps {
  isOpen: boolean;
  onClose: () => void;
  application: Application | null;
  onSuccess?: () => void;
}

export const EditApplicationModal = ({
  isOpen,
  onClose,
  application,
  onSuccess,
}: EditApplicationModalProps) => {
  const [appId, setAppId] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [hostName, setHostName] = useState("");
  const [error, setError] = useState<string | null>(null);

  const { updateApplication, isLoading } = useUpdateApplication();

  useEffect(() => {
    if (application) {
      setAppId(application.appId);
      setDisplayName(application.displayName);
      setHostName(application.hostName);
    } else {
      setAppId("");
      setDisplayName("");
      setHostName("");
    }
    setError(null);
  }, [application, isOpen]);

  const validateAppId = (id: string): string | null => {
    if (!id.trim()) {
      return "App ID is required";
    }
    if (!/^[0-9]{4}-[0-9]{4}$/.test(id)) {
      return "App ID must be in format XXXX-XXXX (e.g., 0001-0003)";
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
    if (!/^[a-zA-Z0-9.-]+$/.test(host)) {
      return "Host name can only contain letters, numbers, dots, and hyphens";
    }
    return null;
  };

  const isSystemApplication = (instanceId: string): boolean => {
    // Check for core system applications that shouldn't be modified
    return instanceId === "MBtskI6D";
  };

  const hasChanges = (): boolean => {
    if (!application) return false;
    return (
      appId !== application.appId ||
      displayName !== application.displayName ||
      hostName !== application.hostName
    );
  };

  const handleSubmit = async () => {
    if (!application) return;

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

    if (!hasChanges()) {
      onClose();
      return;
    }

    const request: UpdateApplicationRequest = {
      instanceId: application.instanceId,
      appId: appId.trim(),
      displayName: displayName.trim(),
      hostName: hostName.trim(),
    };

    const result = await updateApplication(request);

    if (result.success) {
      onClose();
      onSuccess?.();
    } else {
      setError(result.error || "Failed to update application");
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
    validateHostName(hostName) === null &&
    hasChanges() &&
    !isSystemApplication(application?.instanceId || "");

  const isSystemApp =
    application && isSystemApplication(application.instanceId);

  return (
    <DialogRoot open={isOpen} onOpenChange={(e) => !e.open && handleClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit Application</DialogTitle>
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

            {isSystemApp && (
              <Alert.Root status="warning">
                <Alert.Indicator />
                <Alert.Content>
                  <Alert.Title>System Application</Alert.Title>
                  <Alert.Description>
                    This is a core system application. Modifications are
                    restricted to prevent system issues.
                  </Alert.Description>
                </Alert.Content>
              </Alert.Root>
            )}

            <VStack gap={2} align="stretch">
              <Text fontWeight="medium">Instance ID</Text>
              <Text fontFamily="mono" color="gray.600" fontSize="sm">
                {application?.instanceId}
              </Text>
            </VStack>

            <VStack gap={2} align="stretch">
              <Text fontWeight="medium">App ID</Text>
              <Input
                value={appId}
                onChange={(e) => setAppId(e.target.value)}
                placeholder="e.g., 0001-0003"
                disabled={isLoading || isSystemApp || false}
                fontFamily="mono"
              />
              <Text fontSize="xs" color="gray.500">
                Must be in format XXXX-XXXX (e.g., 0001-0003)
              </Text>
            </VStack>

            <VStack gap={2} align="stretch">
              <Text fontWeight="medium">Display Name</Text>
              <Input
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                placeholder="e.g., My Application"
                disabled={isLoading || isSystemApp || false}
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
                disabled={isLoading || isSystemApp || false}
                fontFamily="mono"
              />
              <Text fontSize="xs" color="gray.500">
                Hostname where the application is accessible
              </Text>
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
              disabled={!canSubmit}
            >
              Save Changes
            </Button>
          </HStack>
        </DialogFooter>
      </DialogContent>
    </DialogRoot>
  );
};
