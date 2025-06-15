import {
  Box,
  Button,
  HStack,
  VStack,
  Text,
  Alert,
  DialogRoot,
  DialogContent,
  DialogHeader,
  DialogBody,
  DialogFooter,
  DialogTitle,
  DialogCloseTrigger,
} from "@chakra-ui/react";
import { useState } from "react";
import { LuX, LuTrash2 } from "react-icons/lu";
import {
  useDeleteUserAccessRule,
  type DeleteUserAccessRuleRequest,
} from "../../dataviews/userAccessRuleActions";
import { type UserAccessRule } from "../../dataviews/userAccessRules";

interface DeleteUserAccessRuleModalProps {
  isOpen: boolean;
  onClose: () => void;
  rule: UserAccessRule | null;
  applicationName: string;
  onSuccess: () => void;
}

export const DeleteUserAccessRuleModal = ({
  isOpen,
  onClose,
  rule,
  applicationName,
  onSuccess,
}: DeleteUserAccessRuleModalProps) => {
  const [error, setError] = useState<string | null>(null);
  const { deleteUserAccessRule, isLoading } = useDeleteUserAccessRule();

  const handleClose = () => {
    setError(null);
    onClose();
  };

  const handleDelete = () => {
    if (!rule) {
      setError("No rule to delete");
      return;
    }

    const request: DeleteUserAccessRuleRequest = {
      ruleId: rule.ruleId,
    };

    deleteUserAccessRule(request);
    onSuccess();
  };

  const getRuleTypeLabel = (ruleType: string) => {
    return ruleType === "ACCEPT" ? "Allow Access" : "Deny Access";
  };

  const getSubjectLabel = (rule: UserAccessRule) => {
    if (rule.subjectType === "USER") {
      return `User ID: ${rule.subjectId}`;
    }
    return `Group: ${rule.subjectId}`;
  };

  if (!rule) {
    return null;
  }

  return (
    <DialogRoot open={isOpen} onOpenChange={(e) => !e.open && handleClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete Access Rule</DialogTitle>
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
                  This action cannot be undone. The access rule will be
                  permanently removed.
                </Alert.Description>
              </Alert.Content>
            </Alert.Root>

            <Box>
              <Text fontSize="sm" color="gray.600" mb={3}>
                You are about to delete the following access rule for{" "}
                <strong>{applicationName}</strong>:
              </Text>

              <VStack
                gap={2}
                align="stretch"
                p={4}
                bg="gray.50"
                borderRadius="md"
              >
                <HStack justify="space-between">
                  <Text fontSize="sm" fontWeight="medium" color="gray.700">
                    Rule Type:
                  </Text>
                  <Text fontSize="sm" color="gray.900">
                    {getRuleTypeLabel(rule.ruleType)}
                  </Text>
                </HStack>
                <HStack justify="space-between">
                  <Text fontSize="sm" fontWeight="medium" color="gray.700">
                    Subject:
                  </Text>
                  <Text fontSize="sm" color="gray.900">
                    {getSubjectLabel(rule)}
                  </Text>
                </HStack>
              </VStack>
            </Box>

            <Box>
              <Text fontSize="sm" color="gray.600">
                Are you sure you want to delete this access rule? This action
                will immediately affect user access to the application.
              </Text>
            </Box>

            {error && (
              <Alert.Root status="error">
                <Alert.Indicator />
                <Alert.Content>
                  <Alert.Description>{error}</Alert.Description>
                </Alert.Content>
              </Alert.Root>
            )}
          </VStack>
        </DialogBody>
        <DialogFooter>
          <HStack gap={2}>
            <Button
              variant="outline"
              onClick={handleClose}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button
              colorScheme="red"
              onClick={handleDelete}
              loading={isLoading}
              disabled={isLoading}
            >
              <LuTrash2 />
              Delete Access Rule
            </Button>
          </HStack>
        </DialogFooter>
      </DialogContent>
    </DialogRoot>
  );
};
