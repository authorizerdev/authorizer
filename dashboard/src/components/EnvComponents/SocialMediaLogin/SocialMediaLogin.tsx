import React from "react";
import { FaGoogle, FaGithub, FaFacebookF } from "react-icons/fa";
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
import { TextInputType, HiddenInputType } from "../../../constants";
import InputField from "../../InputField";
const SocialMediaLogin = ({
  variables,
  setVariables,
  fieldVisibility,
  setFieldVisibility,
}: any) => {
  return (
    <div>
      {" "}
      <Text fontSize="md" paddingTop="2%" fontWeight="bold">
        Social Media Logins
      </Text>
      <Stack spacing={6} padding="2% 0%">
        <Flex>
          <Center
            w="50px"
            marginRight="1.5%"
            border="1px solid #e2e8f0"
            borderRadius="5px"
          >
            <FaGoogle style={{ color: "#8c8c8c" }} />
          </Center>
          <Center w="45%" marginRight="1.5%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={TextInputType.GOOGLE_CLIENT_ID}
              placeholder="Google Client ID"
            />
          </Center>
          <Center w="45%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              fieldVisibility={fieldVisibility}
              setFieldVisibility={setFieldVisibility}
              inputType={HiddenInputType.GOOGLE_CLIENT_SECRET}
              placeholder="Google Secret"
            />
          </Center>
        </Flex>
        <Flex>
          <Center
            w="50px"
            marginRight="1.5%"
            border="1px solid #e2e8f0"
            borderRadius="5px"
          >
            <FaGithub style={{ color: "#8c8c8c" }} />
          </Center>
          <Center w="45%" marginRight="1.5%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={TextInputType.GITHUB_CLIENT_ID}
              placeholder="Github Client ID"
            />
          </Center>
          <Center w="45%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              fieldVisibility={fieldVisibility}
              setFieldVisibility={setFieldVisibility}
              inputType={HiddenInputType.GITHUB_CLIENT_SECRET}
              placeholder="Github Secret"
            />
          </Center>
        </Flex>
        <Flex>
          <Center
            w="50px"
            marginRight="1.5%"
            border="1px solid #e2e8f0"
            borderRadius="5px"
          >
            <FaFacebookF style={{ color: "#8c8c8c" }} />
          </Center>
          <Center w="45%" marginRight="1.5%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={TextInputType.FACEBOOK_CLIENT_ID}
              placeholder="Facebook Client ID"
            />
          </Center>
          <Center w="45%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              fieldVisibility={fieldVisibility}
              setFieldVisibility={setFieldVisibility}
              inputType={HiddenInputType.FACEBOOK_CLIENT_SECRET}
              placeholder="Facebook Secret"
            />
          </Center>
        </Flex>
      </Stack>
    </div>
  );
};

export default SocialMediaLogin;
