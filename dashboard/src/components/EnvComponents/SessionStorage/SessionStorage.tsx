import React from "react";
import {
  Box,
  Divider,
  Flex,
  Stack,
  Center,
  Text,
  Button,
  Input,
  InputGroup,
  InputRightElement,
  useToast,
} from "@chakra-ui/react";
import InputField from "../../InputField";
const SessionStorage = ({ variables, setVariables, RedisURL }: any) => {
  return (
    <div>
      {" "}
      <Text fontSize="md" paddingTop="2%" fontWeight="bold">
        Session Storage
      </Text>
      <Stack spacing={6} padding="2% 0%">
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Redis URL:</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={RedisURL}
            />
          </Center>
        </Flex>
      </Stack>
    </div>
  );
};

export default SessionStorage;
