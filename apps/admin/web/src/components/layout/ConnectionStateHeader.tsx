import { Box, Text, Badge, HStack } from "@chakra-ui/react";
import { useConnectionState } from "@tomyedwab/yesterday";

export const ConnectionStateHeader = () => {
  const connectionState = useConnectionState();

  return (
    <Box bg="gray.50" borderBottom="1px" borderColor="gray.200" p={4}>
      <HStack justify="space-between" align="center">
        <Text fontSize="lg" fontWeight="bold">
          Yesterday Admin
        </Text>
        <HStack gap={4}>
          <Text fontSize="sm" color="gray.600">
            Server: {connectionState.serverVersion}
          </Text>
          <Badge
            colorScheme={connectionState.connected ? "green" : "red"}
            variant="subtle"
          >
            {connectionState.connected ? "Connected" : "Disconnected"}
          </Badge>
        </HStack>
      </HStack>
    </Box>
  );
};