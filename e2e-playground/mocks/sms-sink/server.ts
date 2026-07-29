import express from 'express';

const app = express();
app.use(express.json());

const latestByPhone = new Map<string, { phone: string; message: string }>();

app.post('/sms', (req, res) => {
  const { phone, message } = req.body;
  latestByPhone.set(phone, { phone, message });
  res.sendStatus(204);
});

app.get('/sms/:phone', (req, res) => {
  const entry = latestByPhone.get(req.params.phone);
  if (!entry) {
    res.sendStatus(404);
    return;
  }
  res.json(entry);
});

app.listen(4100, () => console.log('sms-sink listening on :4100'));
