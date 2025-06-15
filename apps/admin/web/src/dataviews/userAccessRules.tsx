import { useDataView } from "@tomyedwab/yesterday";

export type RuleType = "ACCEPT" | "DENY";
export type SubjectType = "USER" | "GROUP";

export interface UserAccessRule {
  ruleId: string;
  applicationId: string;
  ruleType: RuleType;
  subjectType: SubjectType;
  subjectId: string;
  createdAt: string;
  updatedAt: string;
}

export interface UserAccessRulesViewResponse {
  rules: UserAccessRule[];
  total: number;
}

/**
 * Hook to fetch user access rules for a specific application
 * @param applicationId - The application ID to fetch rules for
 * @returns [loading, rules] tuple
 */
export function useUserAccessRulesView(
  applicationId: string,
): [boolean, UserAccessRule[]] {
  const params = { applicationId };
  const [loading, response] = useDataView("api/user-access-rules", params);

  if (loading || response === null) {
    return [true, []];
  }

  return [false, response.rules || []];
}

/**
 * Hook to fetch all user access rules across all applications
 * @returns [loading, rules] tuple
 */
export function useAllUserAccessRulesView(): [boolean, UserAccessRule[]] {
  const [loading, response] = useDataView("api/user-access-rules");

  if (loading || response === null) {
    return [true, []];
  }

  return [false, response.rules || []];
}
