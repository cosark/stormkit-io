import { useContext } from "react";
import Avatar from "@mui/material/Avatar";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { AccountContext } from "../Account.context";

export default function Instances() {
  const { account, signOut } = useContext(AccountContext);

  return (
    <Box sx={{ maxWidth: 960, width: "100%", mx: "auto", p: 4 }}>
      <Stack direction="row" alignItems="center" spacing={2} sx={{ mb: 4 }}>
        <Avatar src={account?.avatarUrl} alt={account?.displayName} />
        <Typography sx={{ flex: 1 }}>{account?.displayName}</Typography>
        <Button onClick={signOut}>Sign out</Button>
      </Stack>
      <Typography variant="h5" component="h1" sx={{ mb: 2 }}>
        Your instances
      </Typography>
      <Paper variant="outlined" sx={{ p: 4, textAlign: "center" }}>
        <Typography color="text.secondary">
          You have no instances yet.
        </Typography>
      </Paper>
    </Box>
  );
}
