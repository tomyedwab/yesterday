import { Box, Tabs } from "@chakra-ui/react";
import { LuUsers, LuCog } from "react-icons/lu";
import { UsersTab } from "../users/UsersTab";
import { ConnectionStateHeader } from "./ConnectionStateHeader";

export const MainLayout = () => {
  return (
    <Box>
      <ConnectionStateHeader />
      <Box p={6}>
        <Tabs.Root defaultValue="users" variant="enclosed">
          <Tabs.List>
            <Tabs.Trigger value="users">
              <LuUsers />
              Users
            </Tabs.Trigger>
            <Tabs.Trigger value="applications">
              <LuCog />
              Applications
            </Tabs.Trigger>
          </Tabs.List>
          <Tabs.Content value="users">
            <UsersTab />
          </Tabs.Content>
          <Tabs.Content value="applications">
            <Box p={4}>
              <p>Applications management coming soon...</p>
            </Box>
          </Tabs.Content>
        </Tabs.Root>
      </Box>
    </Box>
  );
};