import { useDataView } from "@tomyedwab/yesterday";

export type User = {
  id: number;
  username: string;
};

export function useUsersView(): [boolean, User[]] {
  const [loading, response] = useDataView("api/users");
  if (loading || response === null) {
    return [true, []];
  }
  return [false, response.users];
}
