import {
  useConnectionDispatch,
  CreatePendingEvent,
} from "@tomyedwab/yesterday";
import type { RuleType, SubjectType } from "./userAccessRules";

export interface CreateUserAccessRuleRequest {
  applicationId: string;
  ruleType: RuleType;
  subjectType: SubjectType;
  subjectId: string;
}

export interface UpdateUserAccessRuleRequest {
  ruleId: string;
  ruleType: RuleType;
  subjectType: SubjectType;
  subjectId: string;
}

export interface DeleteUserAccessRuleRequest {
  ruleId: string;
}

/**
 * Hook for creating user access rules
 */
export function useCreateUserAccessRule() {
  const connectDispatch = useConnectionDispatch();

  const createUserAccessRule = (request: CreateUserAccessRuleRequest) => {
    connectDispatch(
      CreatePendingEvent("createuseraccessrule:", {
        type: "CreateUserAccessRule",
        applicationId: request.applicationId,
        ruleType: request.ruleType,
        subjectType: request.subjectType,
        subjectId: request.subjectId,
      }),
    );
  };

  return { createUserAccessRule, isLoading: false };
}

/**
 * Hook for updating user access rules
 */
export function useUpdateUserAccessRule() {
  const connectDispatch = useConnectionDispatch();

  const updateUserAccessRule = (request: UpdateUserAccessRuleRequest) => {
    connectDispatch(
      CreatePendingEvent("updateuseraccessrule:", {
        type: "UpdateUserAccessRule",
        ruleId: request.ruleId,
        ruleType: request.ruleType,
        subjectType: request.subjectType,
        subjectId: request.subjectId,
      }),
    );
  };

  return { updateUserAccessRule, isLoading: false };
}

/**
 * Hook for deleting user access rules
 */
export function useDeleteUserAccessRule() {
  const connectDispatch = useConnectionDispatch();

  const deleteUserAccessRule = (request: DeleteUserAccessRuleRequest) => {
    connectDispatch(
      CreatePendingEvent("deleteuseraccessrule:", {
        type: "DeleteUserAccessRule",
        ruleId: request.ruleId,
      }),
    );
  };

  return { deleteUserAccessRule, isLoading: false };
}
