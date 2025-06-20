import { useDataView } from "@tomyedwab/yesterday";

export type Application = {
  instanceId: string;
  appId: string;
  displayName: string;
  hostName: string;
};

export function useApplicationsView(): [boolean, Application[]] {
  const [loading, response] = useDataView("api/applications");
  if (loading || response === null) {
    return [true, []];
  }
  return [false, response.applications];
}
