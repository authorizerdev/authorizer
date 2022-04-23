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
import { ArrayInputType } from "../../../constants";
import InputField from "../../InputField";

const Roles = ({ variables, setVariables }: any) => {
  return (
    <div>
      {" "}
      <Text fontSize="md" paddingTop="2%" fontWeight="bold">
        Roles
      </Text>
      <Stack spacing={6} padding="2% 0%">
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Roles:</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={ArrayInputType.ROLES}
            />
          </Center>
        </Flex>
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Default Roles:</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={ArrayInputType.DEFAULT_ROLES}
            />
          </Center>
        </Flex>
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Protected Roles:</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={ArrayInputType.PROTECTED_ROLES}
            />
          </Center>
        </Flex>
      </Stack>
    </div>
  );
};

export default Roles;
