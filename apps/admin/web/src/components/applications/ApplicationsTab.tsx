import { Box, Heading, VStack } from "@chakra-ui/react";
import { ApplicationsList } from "./ApplicationsList";

export const ApplicationsTab = () => {
  return (
    <Box p={4}>
      <VStack align="stretch" gap={6}>
        <Heading size="lg">Application Management</Heading>
        <ApplicationsList />
      </VStack>
    </Box>
  );
};