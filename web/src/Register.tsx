import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import axios from "axios";
import { toast, ToastContainer } from "react-toastify";
import "react-toastify/dist/ReactToastify.css";
import { Button } from "./components/ui/button";
import { Input } from "./components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "./components/ui/card";
import { Eye, EyeOff } from "lucide-react";

const Register: React.FC = () => {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const navigate = useNavigate();

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();

    // Basic validation
    if (!email || !password) {
      toast.error("Please fill out all required fields");
      return;
    }

    if (password !== confirmPassword) {
      toast.error("Passwords do not match");
      return;
    }

    if (password.length < 8) {
      toast.error("Password must be at least 8 characters long");
      return;
    }

    setLoading(true);

    try {
      await axios.post("/api/register", {
        name,
        email,
        password,
      });

      toast.success("Registration successful! Please log in.");
      navigate("/login");
    } catch (error: any) {
      console.error("Registration failed:", error);

      if (error.response?.status === 409) {
        toast.error("Email already in use. Please use a different email.");
      } else {
        toast.error("Registration failed. Please try again.");
      }
    } finally {
      setLoading(false);
    }
  };

  const togglePasswordVisibility = () => {
    setShowPassword(!showPassword);
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-hyper-gray">
      <Card className="w-full max-w-md shadow-lg">
        <CardHeader className="text-center">
          <h1 className="text-3xl font-bold text-hyper-blue">NyatiCtl</h1>
          <CardTitle className="text-2xl mt-2 text-hyper-blue">
            Register
          </CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleRegister} className="space-y-4">
            <div>
              <label
                htmlFor="name"
                className="block text-sm font-medium text-gray-700"
              >
                Name
              </label>
              <Input
                id="name"
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Enter your name"
                className="mt-1"
                disabled={loading}
              />
            </div>

            <div>
              <label
                htmlFor="email"
                className="block text-sm font-medium text-gray-700"
              >
                Email
              </label>
              <Input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="Enter your email"
                className="mt-1"
                disabled={loading}
                required
              />
            </div>

            <div>
              <label
                htmlFor="password"
                className="block text-sm font-medium text-gray-700"
              >
                Password
              </label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Enter your password"
                  className="mt-1 pr-10"
                  disabled={loading}
                  required
                />
                <button
                  type="button"
                  onClick={togglePasswordVisibility}
                  className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400"
                >
                  {showPassword ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </button>
              </div>
            </div>

            <div>
              <label
                htmlFor="confirmPassword"
                className="block text-sm font-medium text-gray-700"
              >
                Confirm Password
              </label>
              <Input
                id="confirmPassword"
                type={showPassword ? "text" : "password"}
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="Confirm your password"
                className="mt-1"
                disabled={loading}
                required
              />
            </div>

            <Button
              type="submit"
              className="w-full bg-hyper-cyan hover:bg-hyper-cyan/90"
              disabled={loading}
            >
              {loading ? "Registering..." : "Register"}
            </Button>

            <div className="text-center">
              <div className="text-sm text-gray-600">
                Already have an account?{" "}
                <Link to="/login" className="text-hyper-blue hover:underline">
                  Login
                </Link>
              </div>
            </div>
          </form>
        </CardContent>
      </Card>
      <ToastContainer />
    </div>
  );
};

export default Register;
