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
import { TextInputType } from "../../../constants";

const OrganizationInfo = ({ variables, setVariables }: any) => {
  return (
    <div>
      {" "}
      <Text fontSize="md" paddingTop="2%" fontWeight="bold">
        Organization Information
      </Text>
      <Stack spacing={6} padding="2% 0%">
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Organization Name:</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={TextInputType.ORGANIZATION_NAME}
            />
          </Center>
        </Flex>
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Organization Logo:</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={TextInputType.ORGANIZATION_LOGO}
            />
          </Center>
        </Flex>
      </Stack>
    </div>
  );
};

export default OrganizationInfo;
