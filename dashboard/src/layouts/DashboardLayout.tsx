import { Box, Flex } from "@chakra-ui/react";
import React from "react";
import { Sidebar } from "../components/Sidebar";

export function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <Flex flexWrap="wrap" h="100%">
      <Box maxW="72" bg="blue.500" flex="1">
        <Sidebar />
      </Box>
      <Box as="main" flex="2" p="10">{children}</Box>
    </Flex>
  );
}
