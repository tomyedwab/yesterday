import { Box, Heading, VStack } from "@chakra-ui/react";
import { UsersList } from "./UsersList";

export const UsersTab = () => {
  return (
    <Box p={4}>
      <VStack align="stretch" gap={6}>
        <Heading size="lg">User Management</Heading>
        <UsersList />
      </VStack>
    </Box>
  );
};