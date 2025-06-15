import {
  Box,
  Button,
  HStack,
  VStack,
  Text,
  Input,
  NativeSelectField,
  NativeSelectRoot,
  Alert,
  DialogRoot,
  DialogContent,
  DialogHeader,
  DialogBody,
  DialogFooter,
  DialogTitle,
  DialogCloseTrigger,
  Spinner,
} from "@chakra-ui/react";
import { useState, useEffect } from "react";
import { LuX } from "react-icons/lu";
import {
  useUpdateUserAccessRule,
  type UpdateUserAccessRuleRequest,
} from "../../dataviews/userAccessRuleActions";
import {
  type RuleType,
  type SubjectType,
  type UserAccessRule,
} from "../../dataviews/userAccessRules";
import { useUsersView } from "../../dataviews/users";

interface EditUserAccessRuleModalProps {
  isOpen: boolean;
  onClose: () => void;
  rule: UserAccessRule | null;
  applicationName: string;
  onSuccess: () => void;
}

export const EditUserAccessRuleModal = ({
  isOpen,
  onClose,
  rule,
  applicationName,
  onSuccess,
}: EditUserAccessRuleModalProps) => {
  const [ruleType, setRuleType] = useState<RuleType>("ACCEPT");
  const [subjectType, setSubjectType] = useState<SubjectType>("USER");
  const [subjectId, setSubjectId] = useState("");
  const [error, setError] = useState<string | null>(null);

  const { updateUserAccessRule, isLoading } = useUpdateUserAccessRule();
  const [usersLoading, users] = useUsersView();

  // Reset form when modal opens or rule changes
  useEffect(() => {
    if (isOpen && rule) {
      setRuleType(rule.ruleType);
      setSubjectType(rule.subjectType);
      setSubjectId(rule.subjectId);
      setError(null);
    }
  }, [isOpen, rule]);

  const handleClose = () => {
    setError(null);
    onClose();
  };

  const handleSubmit = () => {
    if (!rule) {
      setError("No rule to update");
      return;
    }

    if (!subjectId.trim()) {
      setError("Subject ID is required");
      return;
    }

    // If USER type is selected, validate that the user exists
    if (subjectType === "USER" && !usersLoading) {
      const userExists = users.some(
        (user) => user.id.toString() === subjectId.trim(),
      );
      if (!userExists) {
        setError("User does not exist. Please select a valid user ID.");
        return;
      }
    }

    const request: UpdateUserAccessRuleRequest = {
      ruleId: rule.ruleId,
      ruleType,
      subjectType,
      subjectId: subjectId.trim(),
    };

    updateUserAccessRule(request);
    onSuccess();
  };

  if (!rule) {
    return null;
  }

  return (
    <DialogRoot open={isOpen} onOpenChange={(e) => !e.open && handleClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit Access Rule</DialogTitle>
          <DialogCloseTrigger asChild>
            <Button variant="ghost" size="sm" disabled={isLoading}>
              <LuX />
            </Button>
          </DialogCloseTrigger>
        </DialogHeader>
        <DialogBody>
          <VStack gap={4} align="stretch">
            <Box>
              <Text fontSize="sm" color="gray.600" mb={3}>
                Edit access rule for <strong>{applicationName}</strong>
              </Text>
            </Box>

            {/* Rule Type Selection */}
            <VStack gap={2} align="stretch">
              <Text fontWeight="medium">Rule Type</Text>
              <NativeSelectRoot>
                <NativeSelectField
                  value={ruleType}
                  onChange={(e) => setRuleType(e.target.value as RuleType)}
                >
                  <option value="ACCEPT">Allow Access</option>
                  <option value="DENY">Deny Access</option>
                </NativeSelectField>
              </NativeSelectRoot>
            </VStack>

            {/* Subject Type Selection */}
            <VStack gap={2} align="stretch">
              <Text fontWeight="medium">Subject Type</Text>
              <NativeSelectRoot>
                <NativeSelectField
                  value={subjectType}
                  onChange={(e) =>
                    setSubjectType(e.target.value as SubjectType)
                  }
                >
                  <option value="USER">User</option>
                  <option value="GROUP">Group</option>
                </NativeSelectField>
              </NativeSelectRoot>
            </VStack>

            {/* Subject ID Input */}
            <VStack gap={2} align="stretch">
              <Text fontWeight="medium">
                {subjectType === "USER" ? "User" : "Group Name"}
              </Text>
              {subjectType === "USER" && !usersLoading ? (
                <NativeSelectRoot>
                  <NativeSelectField
                    value={subjectId}
                    onChange={(e) => setSubjectId(e.target.value)}
                  >
                    <option value="">Select a user</option>
                    {users.map((user) => (
                      <option key={user.id} value={user.id.toString()}>
                        {user.username} (ID: {user.id})
                      </option>
                    ))}
                  </NativeSelectField>
                </NativeSelectRoot>
              ) : subjectType === "USER" && usersLoading ? (
                <HStack gap={2} p={2}>
                  <Spinner size="sm" />
                  <Text fontSize="sm" color="gray.600">
                    Loading users...
                  </Text>
                </HStack>
              ) : (
                <Input
                  placeholder={
                    subjectType === "USER"
                      ? "Enter user ID"
                      : "Enter group name"
                  }
                  value={subjectId}
                  onChange={(e) => setSubjectId(e.target.value)}
                  disabled={isLoading}
                />
              )}
            </VStack>

            {/* Error Display */}
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
              colorScheme="blue"
              onClick={handleSubmit}
              disabled={isLoading || !subjectId.trim()}
              loading={isLoading}
            >
              Update Access Rule
            </Button>
          </HStack>
        </DialogFooter>
      </DialogContent>
    </DialogRoot>
  );
};
