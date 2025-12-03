import express from "express";

const app = express();

app.use(express.json());


app.get("/", (req, res) => {
    res.json({ message: "Server is running" });
});

app.listen(3005, () => {
    console.log("Express server running on port 3005");
});
