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
import { Button, VStack, Text, Alert, HStack } from "@chakra-ui/react";
import { LuX, LuTrash2 } from "react-icons/lu";
import {
  useDeleteApplication,
  type DeleteApplicationRequest,
} from "../../dataviews/applicationActions";
import type { Application } from "../../dataviews/applications";

interface DeleteApplicationModalProps {
  isOpen: boolean;
  onClose: () => void;
  application: Application | null;
  onSuccess?: () => void;
}

export const DeleteApplicationModal = ({
  isOpen,
  onClose,
  application,
  onSuccess,
}: DeleteApplicationModalProps) => {
  const [error, setError] = useState<string | null>(null);
  const { deleteApplication, isLoading } = useDeleteApplication();

  useEffect(() => {
    if (isOpen) {
      setError(null);
    }
  }, [isOpen]);

  const isSystemApplication = (instanceId: string): boolean => {
    // Check for core system applications that shouldn't be deleted
    return instanceId === "MBtskI6D";
  };

  const handleSubmit = async () => {
    if (!application) return;

    if (isSystemApplication(application.instanceId)) {
      setError("Cannot delete core system applications");
      return;
    }

    const request: DeleteApplicationRequest = {
      instanceId: application.instanceId,
    };

    const result = await deleteApplication(request);

    if (result.success) {
      onClose();
      onSuccess?.();
    } else {
      setError(result.error || "Failed to delete application");
    }
  };

  const handleClose = () => {
    if (!isLoading) {
      onClose();
    }
  };

  const isSystemApp =
    application && isSystemApplication(application.instanceId);

  return (
    <DialogRoot open={isOpen} onOpenChange={(e) => !e.open && handleClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete Application</DialogTitle>
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

            {isSystemApp ? (
              <Alert.Root status="error">
                <Alert.Indicator />
                <Alert.Content>
                  <Alert.Title>Cannot Delete System Application</Alert.Title>
                  <Alert.Description>
                    This is a core system application and cannot be deleted to
                    prevent system issues.
                  </Alert.Description>
                </Alert.Content>
              </Alert.Root>
            ) : (
              <Alert.Root status="warning">
                <Alert.Indicator />
                <Alert.Content>
                  <Alert.Title>Confirm Deletion</Alert.Title>
                  <Alert.Description>
                    This action cannot be undone. All associated user access
                    rules will also be deleted.
                  </Alert.Description>
                </Alert.Content>
              </Alert.Root>
            )}

            <VStack
              gap={3}
              align="stretch"
              p={4}
              bg="gray.50"
              borderRadius="md"
            >
              <Text fontWeight="medium" fontSize="sm">
                Application Details:
              </Text>

              <HStack justify="space-between">
                <Text fontSize="sm" color="gray.600">
                  Display Name:
                </Text>
                <Text fontSize="sm" fontWeight="medium">
                  {application?.displayName}
                </Text>
              </HStack>

              <HStack justify="space-between">
                <Text fontSize="sm" color="gray.600">
                  App ID:
                </Text>
                <Text fontSize="sm" fontFamily="mono">
                  {application?.appId}
                </Text>
              </HStack>

              <HStack justify="space-between">
                <Text fontSize="sm" color="gray.600">
                  Host Name:
                </Text>
                <Text fontSize="sm" fontFamily="mono">
                  {application?.hostName}
                </Text>
              </HStack>

              <HStack justify="space-between">
                <Text fontSize="sm" color="gray.600">
                  Instance ID:
                </Text>
                <Text fontSize="sm" fontFamily="mono" color="gray.500">
                  {application?.instanceId}
                </Text>
              </HStack>
            </VStack>

            {!isSystemApp && (
              <Text fontSize="sm" color="gray.600">
                Are you sure you want to delete this application? This will also
                remove all associated user access rules.
              </Text>
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
            {!isSystemApp && (
              <Button
                colorScheme="red"
                onClick={handleSubmit}
                loading={isLoading}
                disabled={isLoading}
              >
                <LuTrash2 />
                Delete Application
              </Button>
            )}
          </HStack>
        </DialogFooter>
      </DialogContent>
    </DialogRoot>
  );
};
